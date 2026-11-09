package org.mygo

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Handler
import android.os.Looper
import org.json.JSONObject
import java.util.UUID

/**
 * 一个远程推送通道，由应用自己实现。MyGo 不自带任何通道。
 *
 * 在 AndroidManifest.xml 的 <application> 里按优先级列出实现类（Kotlin object 或带
 * 无参构造的类），逗号分隔：
 *
 *     <meta-data android:name="org.mygo.push.channels" android:value="com.example.MyChannel" />
 *
 * 通道拿到地址时调用 [MygoPush.subscribed]，失败时调用 [MygoPush.failed]（换下一个通道），
 * 收到消息时调用 [MygoPush.received]。
 */
interface MygoPushChannel {
    val service: String
    fun available(ctx: Context): Boolean
    fun register(ctx: Context)
}

/**
 * 远程推送的通用部分：选通道、把订阅交给 Go、显示远程通知、把点击带回 Go。
 *
 * 消息是 UTF-8 JSON，由本应用显示，所以点击能拿到完整 data，前台时也能先问 Go 要不要弹：
 * {"v":1,"id":"…","title":"…","body":"…","group":"…","badge":3,"silent":false,"data":{…}}
 *
 * 订阅存在 SharedPreferences：通道的回调可能在 Go 还没跑的进程里到来，等下一次 register 时回放。
 */
object MygoPush {
    const val ACTION_CLICK = "org.mygo.PUSH_CLICK"
    private const val META_CHANNELS = "org.mygo.push.channels"
    private const val EXTRA_PAYLOAD = "org.mygo.PUSH_PAYLOAD"
    private const val EXTRA_KEY = "org.mygo.PUSH_KEY"
    private const val CHANNEL = "org.mygo.remote"
    /** 远程通知共用一个 id，按 tag（通知 id）区分；同 id 的新通知替换旧的。 */
    private const val NOTIFICATION_ID = 0x6d79
    private const val PREFS = "org.mygo.push"

    private val main = Handler(Looper.getMainLooper())

    /** Go 已经调用过 register：本进程里有 Go 在听，可以把事件交过去。 */
    @Volatile private var attached = false
    /** 当前使用的通道；其他通道的回调只更新存储，不上报。 */
    @Volatile private var chosen: String? = null
    private var loaded: List<MygoPushChannel>? = null

    private fun channels(ctx: Context): List<MygoPushChannel> {
        loaded?.let { return it }
        val names = try {
            ctx.packageManager.getApplicationInfo(ctx.packageName, PackageManager.GET_META_DATA).metaData?.getString(META_CHANNELS)
        } catch (_: Exception) { null }
        val list = names.orEmpty().split(',').map { it.trim() }.filter { it.isNotEmpty() }.mapNotNull { name ->
            try {
                val c = Class.forName(name)
                (runCatching { c.getField("INSTANCE").get(null) }.getOrNull() ?: c.getDeclaredConstructor().newInstance()) as? MygoPushChannel
            } catch (_: Throwable) { null }
        }
        loaded = list
        return list
    }

    private fun prefs(ctx: Context) = ctx.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    /** Go 的 RegisterPushNotifications（服务请求 push-register，UI 线程）。 */
    fun register(ctx: Context) {
        attached = true
        start(ctx.applicationContext, 0, null)
    }

    private fun start(app: Context, from: Int, lastError: String?) {
        val all = channels(app)
        if (all.isEmpty()) {
            report(JSONObject().put("error", "The application registers no push channel"))
            return
        }
        for (i in from until all.size) {
            val c = all[i]
            if (!runCatching { c.available(app) }.getOrDefault(false)) continue
            chosen = c.service
            prefs(app).getString("sub:" + c.service, null)?.let { saved -> runCatching { report(JSONObject(saved)) } }
            try {
                c.register(app)
            } catch (e: Exception) {
                failed(app, c.service, e.message ?: e.javaClass.simpleName)
            }
            return
        }
        chosen = null
        report(JSONObject().put("error", lastError ?: "No push channel is available on this device"))
    }

    /** 通道拿到了（新的）地址。任何线程。 */
    fun subscribed(ctx: Context, service: String, token: String, data: Map<String, String> = emptyMap()) {
        val sub = JSONObject().put("service", service).put("token", token).put("data", JSONObject(data))
        val p = prefs(ctx)
        val changed = p.getString("sub:$service", null) != sub.toString()
        p.edit().putString("sub:$service", sub.toString()).apply()
        if (service == chosen && changed) report(sub)
    }

    /** 通道的注册被撤销。 */
    fun unsubscribed(ctx: Context, service: String) {
        prefs(ctx).edit().remove("sub:$service").apply()
        if (service == chosen) report(JSONObject().put("error", "The $service registration was removed"))
    }

    /** 当前通道注册失败：换下一个可用通道。 */
    fun failed(ctx: Context, service: String, message: String) {
        if (service != chosen) return
        val i = channels(ctx).indexOfFirst { it.service == service }
        main.post { start(ctx.applicationContext, if (i < 0) Int.MAX_VALUE else i + 1, message) }
    }

    private fun report(json: JSONObject) {
        if (attached) Native.onPush(json.toString().toByteArray(Charsets.UTF_8))
    }

    /** 通道收到一条消息（JSON）。任何线程。 */
    fun received(ctx: Context, raw: String) {
        val p = try { JSONObject(raw) } catch (_: Exception) { return }
        if (p.optInt("v", 1) != 1) return
        val app = ctx.applicationContext
        main.post {
            if (p.optBoolean("silent")) {
                // 纯角标更新：清零表示全部已读，收起已显示的远程通知。
                if (p.has("badge") && p.optInt("badge") == 0) clearAll(app)
                return@post
            }
            if (attached && MygoServices.isForeground()) {
                // 前台：先交给 Go（presentation handler），它决定后经 push-present 交回来显示。
                val event = JSONObject().put("id", p.optString("id")).put("data", p.optJSONObject("data") ?: JSONObject())
                    .put("clicked", false).put("source", "remote").put("present", p)
                Native.onNotification(event.toString().toByteArray(Charsets.UTF_8))
            } else {
                show(app, p)
            }
        }
    }

    private fun clickKey(ctx: Context): String {
        val p = prefs(ctx)
        return p.getString("click-key", null) ?: UUID.randomUUID().toString().also { p.edit().putString("click-key", it).apply() }
    }

    /** 显示一条远程通知（服务请求 push-present，或后台收到时）。 */
    fun show(ctx: Context, p: JSONObject) {
        val nm = ctx.getSystemService(NotificationManager::class.java) ?: return
        if (!nm.areNotificationsEnabled()) return
        if (nm.getNotificationChannel(CHANNEL) == null) {
            val res = ctx.resources.getIdentifier("mygo_push_channel", "string", ctx.packageName)
            val name = if (res != 0) ctx.getString(res) else "Notifications"
            nm.createNotificationChannel(NotificationChannel(CHANNEL, name, NotificationManager.IMPORTANCE_HIGH).apply { setShowBadge(true) })
        }
        val id = p.optString("id").ifEmpty { "remote" }
        // 直接启动 Activity：Android 12 起不允许从 BroadcastReceiver 跳板启动 Activity。
        // 请求码按 id 区分，否则不同通知的 PendingIntent 会被视为同一个（extras 不参与比较）。
        val intent = Intent(ctx, MygoActivity::class.java).setAction(ACTION_CLICK)
            .putExtra(EXTRA_PAYLOAD, p.toString()).putExtra(EXTRA_KEY, clickKey(ctx))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        val pending = PendingIntent.getActivity(ctx, id.hashCode(), intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val title = p.optString("title")
        val body = p.optString("body")
        val icon = ctx.applicationInfo.icon.takeIf { it != 0 } ?: android.R.drawable.ic_dialog_info
        val b = Notification.Builder(ctx, CHANNEL).setSmallIcon(icon).setContentTitle(title).setContentText(body)
            .setStyle(Notification.BigTextStyle().bigText(body)).setAutoCancel(true).setContentIntent(pending)
            .setWhen(System.currentTimeMillis()).setShowWhen(true)
        p.optString("group").takeIf { it.isNotEmpty() }?.let { b.setGroup(it) }
        if (p.has("badge")) b.setNumber(p.optInt("badge"))
        nm.notify(id, NOTIFICATION_ID, b.build())
    }

    private fun clearAll(ctx: Context) {
        val nm = ctx.getSystemService(NotificationManager::class.java) ?: return
        for (n in nm.activeNotifications) if (n.id == NOTIFICATION_ID) nm.cancel(n.tag, n.id)
    }

    /**
     * MygoActivity 的 onCreate / onNewIntent：点击远程通知启动的 intent。
     * MygoActivity 是导出的（launcher），所以核对本机随机密钥，别的应用伪造不了点击。
     */
    fun handleIntent(ctx: Context, intent: Intent?) {
        if (intent?.action != ACTION_CLICK) return
        val raw = intent.getStringExtra(EXTRA_PAYLOAD)
        val key = intent.getStringExtra(EXTRA_KEY)
        intent.action = null
        intent.removeExtra(EXTRA_PAYLOAD)
        intent.removeExtra(EXTRA_KEY)
        if (raw == null || key != clickKey(ctx)) return
        val p = try { JSONObject(raw) } catch (_: Exception) { return }
        val event = JSONObject().put("id", p.optString("id")).put("data", p.optJSONObject("data") ?: JSONObject())
            .put("clicked", true).put("source", "remote")
        Native.onNotification(event.toString().toByteArray(Charsets.UTF_8))
    }
}
