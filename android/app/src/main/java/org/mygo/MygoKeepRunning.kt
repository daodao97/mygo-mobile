package org.mygo

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.util.Log

/**
 * Go 的 Power.KeepRunning（服务请求 keep-running）：value 为说明文字时启动或更新前台服务，
 * 为 null 时停止。前台服务让系统不冻结、不回收本进程，应用自己的连接因此在后台也能继续。
 *
 * · 类型用 specialUse：dataSync 在 Android 15 上每天限 6 小时，不适合长连接。
 * · START_NOT_STICKY：进程被杀后不自动重启 —— Go 侧的应用要等 Activity 起来才 Ready，
 *   单独拉起服务也跑不起应用逻辑。
 * · Android 12 起，应用在后台时不能启动前台服务：应在前台时调用 KeepRunning。
 */
object MygoKeepRunning {
    const val EXTRA_REASON = "org.mygo.KEEP_RUNNING_REASON"

    fun set(ctx: Context, reason: String?) {
        val app = ctx.applicationContext
        val intent = Intent(app, MygoKeepRunningService::class.java)
        if (reason == null) {
            app.stopService(intent)
            return
        }
        try {
            androidx.core.content.ContextCompat.startForegroundService(app, intent.putExtra(EXTRA_REASON, reason))
        } catch (e: Exception) {
            // ForegroundServiceStartNotAllowedException：在后台调用了 KeepRunning。
            Log.w("mygo", "KeepRunning: unable to start the foreground service: $e")
        }
    }
}

class MygoKeepRunningService : Service() {
    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val reason = intent?.getStringExtra(MygoKeepRunning.EXTRA_REASON) ?: ""
        val n = notification(reason)
        try {
            if (Build.VERSION.SDK_INT >= 34) startForeground(NOTIFICATION_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
            else startForeground(NOTIFICATION_ID, n)
        } catch (e: Exception) {
            Log.w("mygo", "KeepRunning: startForeground failed: $e")
            stopSelf()
        }
        return START_NOT_STICKY
    }

    private fun notification(reason: String): Notification {
        val nm = getSystemService(NotificationManager::class.java)
        if (nm.getNotificationChannel(CHANNEL) == null) {
            val res = resources.getIdentifier("mygo_keep_running_channel", "string", packageName)
            val name = if (res != 0) getString(res) else "Background activity"
            nm.createNotificationChannel(NotificationChannel(CHANNEL, name, NotificationManager.IMPORTANCE_LOW).apply {
                setShowBadge(false); setSound(null, null)
            })
        }
        val launch = packageManager.getLaunchIntentForPackage(packageName)
        val open = launch?.let { PendingIntent.getActivity(this, 0, it, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE) }
        val icon = applicationInfo.icon.takeIf { it != 0 } ?: android.R.drawable.ic_dialog_info
        return Notification.Builder(this, CHANNEL).setSmallIcon(icon)
            .setContentTitle(applicationInfo.loadLabel(packageManager)).setContentText(reason)
            .setOngoing(true).setShowWhen(false).setContentIntent(open)
            .setForegroundServiceBehavior(Notification.FOREGROUND_SERVICE_IMMEDIATE)
            .build()
    }

    companion object {
        private const val CHANNEL = "org.mygo.keep-running"
        private const val NOTIFICATION_ID = 0x6b72
    }
}
