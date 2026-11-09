package org.mygo

import android.Manifest
import android.app.Activity
import android.app.KeyguardManager
import android.hardware.biometrics.BiometricManager
import android.hardware.biometrics.BiometricPrompt
import android.os.CancellationSignal
import android.app.NotificationManager
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import java.lang.ref.WeakReference
import org.json.JSONObject

/** Asynchronous system services. All request state belongs to Android's UI thread. */
internal object MygoServices {
    init { System.loadLibrary("mygo") }
    private external fun nativeAttach()
    private val ui = Handler(Looper.getMainLooper())
    private var owner = WeakReference<Activity>(null)
    private val pending = mutableSetOf<Long>()
    private val cancellations = mutableMapOf<Long, () -> Unit>()
    private data class PermissionRequest(val id: Long, val kind: String, val completion: ((String) -> Unit)? = null, val valid: (() -> Boolean)? = null, var owner: Activity? = null)
    private val permissions = java.util.ArrayDeque<PermissionRequest>()
    private var activePermission: PermissionRequest? = null
    private var authentication: Long? = null
    private var authenticationOwner: Activity? = null
    private const val PERMISSION_REQUEST = 23101

    private var lifecycleInstalled = false
    private var startedActivities = 0
    private var background = true
    fun install(activity: Activity) {
        if (!lifecycleInstalled) {
            lifecycleInstalled = true
            activity.application.registerActivityLifecycleCallbacks(object : android.app.Application.ActivityLifecycleCallbacks {
                override fun onActivityCreated(a: Activity, state: android.os.Bundle?) {
                    MygoPickers.created(a)
                    activePermission?.let { request -> request.owner?.let { previous ->
                        if (previous.isChangingConfigurations && previous.javaClass == a.javaClass && previous.taskId == a.taskId) request.owner = a
                    } }
                }
                override fun onActivityStarted(a: Activity) {
                    startedActivities++
                    if (background) { background = false; Native.onLifecycle(Native.LIFECYCLE_START) }
                }
                override fun onActivityResumed(a: Activity) { owner = WeakReference(a); Native.onLifecycle(Native.LIFECYCLE_RESUME); nextPermission() }
                override fun onActivityPaused(a: Activity) { Native.onLifecycle(Native.LIFECYCLE_PAUSE) }
                override fun onActivityStopped(a: Activity) {
                    startedActivities = (startedActivities - 1).coerceAtLeast(0)
                    // A native browser is part of the same application. Leaving
                    // the Go Activity for it must not disconnect its own proxy.
                    if (startedActivities == 0 && !a.isChangingConfigurations) { MygoAudio.background(); enterBackgroundIfNeeded() }
                }
                override fun onActivitySaveInstanceState(a: Activity, state: android.os.Bundle) {}
                override fun onActivityDestroyed(a: Activity) {
                    MygoPickers.destroyed(a); MygoPlatform.scanHostDestroyed(a)
                    if (!a.isChangingConfigurations) permissionOwnerDestroyed(a)
                    if (authenticationOwner === a) authentication?.let { id ->
                        cancellations.remove(id)?.invoke()
                        finish(id, code = "inactive", message = "Authentication Activity was destroyed")
                    }
                }
            })
        }
        owner = WeakReference(activity)
        nativeAttach()
    }

    private fun enterBackgroundIfNeeded() {
        if (startedActivities == 0 && !background) {
            background = true; Native.onLifecycle(Native.LIFECYCLE_STOP)
        }
    }

    fun rasterGlyph(bytes: ByteArray): ByteArray? = MygoGlyphs.raster(bytes)

    fun request(bytes: ByteArray) {
        ui.post {
            val request = JSONObject(String(bytes, Charsets.UTF_8))
            val id = request.getLong("id")
            pending.add(id)
            try {
                val a = owner.get()
                if (a == null || a.isFinishing || a.isDestroyed) {
                    finish(id, code = "inactive", message = "No active Activity")
                } else {
                    dispatch(a, id, request.getString("kind"), request.opt("value"))
                }
            } catch (e: Exception) {
                finish(id, code = "failed", message = e.message ?: e.javaClass.simpleName)
            }
        }
    }

    private fun dispatch(a: Activity, id: Long, kind: String, value: Any?) {
        when (kind) {
            "permission" -> {
                val v = value as JSONObject
                val name = v.getString("kind")
                val names = permissionNames(name)
                if (names == null) {
                    finish(id, code = "unsupported", message = "Unknown permission")
                } else if (!v.optBoolean("request") || permissionStatus(a, name) == "granted" || names.isEmpty()) {
                    finish(id, permissionStatus(a, name))
                } else {
                    permissions.add(PermissionRequest(id, name))
                    nextPermission()
                }
            }
            "haptic" -> { MygoEffects.haptic(a, value as String); finish(id, true) }
            "badge" -> MygoEffects.badge(a, id, (value as Number).toInt())
            "push-register" -> { MygoPush.register(a); finish(id, true) }
            "push-present" -> { MygoPush.show(a.applicationContext, value as JSONObject); finish(id, true) }
            "keep-running" -> { MygoKeepRunning.set(a, value as? String); finish(id, true) }
            "theme-source" -> { MygoEffects.sourceChanged(a, value as String); finish(id, true) }
            "status-bar" -> { val options = value as JSONObject; MygoEffects.statusBar(a, options.getString("style"), options.getBoolean("hidden")); finish(id, true) }
            "audio-start" -> MygoAudio.start(a, id, (value as JSONObject).getLong("stream"))
            "audio-stop", "audio-cancel" -> { MygoAudio.stop((value as JSONObject).getLong("stream"), kind == "audio-cancel"); finish(id, true) }
            "browser" -> MygoBrowserActivity.open(a, id, value as String)
            "photos", "open", "export" -> MygoPickers.start(a, id, kind, value as JSONObject)
            "message" -> MygoPickers.message(a, id, value as JSONObject)
            "auth-query" -> if (Build.VERSION.SDK_INT >= 29) finish(id, biometricStatus(a)) else finish(id, code = "unsupported", message = "Biometric authentication requires Android 10")
            "authenticate" -> if (Build.VERSION.SDK_INT >= 29) authenticate(a, id, value as JSONObject) else finish(id, code = "unsupported", message = "Biometric authentication requires Android 10")
            "settings" -> {
                a.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:" + a.packageName)))
                finish(id, true)
            }
            else -> finish(id, code = "unsupported", message = "Native service $kind is unavailable")
        }
    }

    fun cancel(id: Long) {
        ui.post {
            if (!pending.remove(id)) return@post
            cancellations.remove(id)?.invoke()
            permissions.removeAll { it.id == id }
        }
    }

    private fun finish(id: Long, value: Any? = null, code: String = "", message: String = "") {
        if (!pending.remove(id)) return
        cancellations.remove(id)
        val result = JSONObject().put("id", id).put("value", value ?: JSONObject.NULL)
        if (code.isNotEmpty()) result.put("error", JSONObject().put("code", code).put("message", message))
        Native.onServiceResult(result.toString().toByteArray(Charsets.UTF_8))
    }

    private fun biometricStatus(a: Activity): JSONObject {
        val manager = a.getSystemService(BiometricManager::class.java)
        val result = if (Build.VERSION.SDK_INT >= 30) manager.canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            else manager.canAuthenticate()
        val reason = when (result) {
            BiometricManager.BIOMETRIC_SUCCESS -> ""
            BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED -> "not-enrolled"
            else -> "unavailable"
        }
        val kind = if (a.packageManager.hasSystemFeature(PackageManager.FEATURE_FINGERPRINT)) "touch-id"
            else if (a.packageManager.hasSystemFeature(PackageManager.FEATURE_FACE)) "face-id" else "none"
        val secure = a.getSystemService(KeyguardManager::class.java).isDeviceSecure
        return JSONObject().put("Kind", kind).put("Available", result == BiometricManager.BIOMETRIC_SUCCESS)
            .put("UnavailableReason", reason).put("DeviceAuthenticationAvailable", secure)
    }

    private fun authenticate(a: Activity, id: Long, options: JSONObject) {
        if (authentication != null) { finish(id, code = "busy", message = "Authentication is already in progress"); return }
        if (!a.hasWindowFocus()) { finish(id, code = "inactive", message = "Application is not active"); return }
        val status = biometricStatus(a)
        val allowCredential = options.optBoolean("AllowDevicePasscode")
        if (!status.getBoolean("Available") && !(allowCredential && status.getBoolean("DeviceAuthenticationAvailable"))) {
            finish(id, code = if (allowCredential && !status.getBoolean("DeviceAuthenticationAvailable")) "passcode-not-set"
                else status.getString("UnavailableReason"), message = "Device authentication is unavailable")
            return
        }
        val signal = CancellationSignal()
        val builder = BiometricPrompt.Builder(a).setTitle(a.applicationInfo.loadLabel(a.packageManager))
            .setDescription(options.getString("Reason"))
        if (allowCredential) {
            if (Build.VERSION.SDK_INT >= 30) builder.setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG or BiometricManager.Authenticators.DEVICE_CREDENTIAL)
            else builder.setDeviceCredentialAllowed(true)
        } else {
            builder.setNegativeButton("Cancel", a.mainExecutor) { _, _ ->
                endAuthentication(id)
                finish(id, code = "canceled", message = "Authentication canceled")
            }
        }
        val prompt = builder.build()
        authentication = id
        authenticationOwner = a
        cancellations[id] = { endAuthentication(id); signal.cancel() }
        try {
            prompt.authenticate(signal, a.mainExecutor, object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    endAuthentication(id)
                    finish(id, true)
                }
                override fun onAuthenticationError(error: Int, message: CharSequence) {
                    endAuthentication(id)
                    val code = when (error) {
                        BiometricPrompt.BIOMETRIC_ERROR_CANCELED, BiometricPrompt.BIOMETRIC_ERROR_USER_CANCELED -> "canceled"
                        BiometricPrompt.BIOMETRIC_ERROR_NO_BIOMETRICS -> "not-enrolled"
                        BiometricPrompt.BIOMETRIC_ERROR_LOCKOUT, BiometricPrompt.BIOMETRIC_ERROR_LOCKOUT_PERMANENT -> "locked-out"
                        BiometricPrompt.BIOMETRIC_ERROR_NO_DEVICE_CREDENTIAL -> "passcode-not-set"
                        BiometricPrompt.BIOMETRIC_ERROR_HW_NOT_PRESENT, BiometricPrompt.BIOMETRIC_ERROR_HW_UNAVAILABLE -> "unavailable"
                        else -> "failed"
                    }
                    finish(id, code = code, message = message.toString())
                }
            })
        } catch (e: Exception) {
            cancellations.remove(id)?.invoke()
            finish(id, code = "failed", message = e.message ?: "Authentication failed")
        }
    }

    private fun endAuthentication(id: Long) {
        if (authentication == id) { authentication = null; authenticationOwner = null }
    }
    private fun permissionOwnerDestroyed(a: Activity) {
        val active = activePermission
        if (active?.owner === a) {
            activePermission = null
            if (active.completion != null) active.completion.invoke("denied")
            else finish(active.id, code = "inactive", message = "Permission Activity was destroyed")
        }
        val current = owner.get()
        if (current == null || current === a || current.isDestroyed || current.isFinishing) {
            while (permissions.isNotEmpty()) {
                val request = permissions.removeFirst()
                if (request.completion != null) request.completion.invoke("denied")
                else finish(request.id, code = "inactive", message = "No Activity for permission request")
            }
        } else nextPermission()
    }
    fun onUserLeave() {
        authentication?.let { id ->
            cancellations.remove(id)?.invoke()
            finish(id, code = "inactive", message = "Application entered the background")
        }
    }

    fun complete(id: Long, value: Any? = null, code: String = "", message: String = "") = finish(id, value, code, message)
    fun post(block: () -> Unit) { ui.post(block) }
    fun isForeground() = startedActivities > 0
    fun isPending(id: Long) = pending.contains(id)
    fun onCancel(id: Long, block: () -> Unit) { cancellations[id] = block }

    private fun permissionNames(kind: String): Array<String>? = when (kind) {
        "camera" -> arrayOf(Manifest.permission.CAMERA)
        "microphone", "speech" -> arrayOf(Manifest.permission.RECORD_AUDIO)
        "geolocation" -> arrayOf(Manifest.permission.ACCESS_COARSE_LOCATION, Manifest.permission.ACCESS_FINE_LOCATION)
        "notifications" -> if (Build.VERSION.SDK_INT >= 33) arrayOf(Manifest.permission.POST_NOTIFICATIONS) else emptyArray()
        "photos-add-only" -> if (Build.VERSION.SDK_INT >= 29) emptyArray() else arrayOf(Manifest.permission.WRITE_EXTERNAL_STORAGE)
        "photos" -> if (Build.VERSION.SDK_INT >= 34) {
            arrayOf(Manifest.permission.READ_MEDIA_IMAGES, Manifest.permission.READ_MEDIA_VISUAL_USER_SELECTED)
        } else if (Build.VERSION.SDK_INT >= 33) arrayOf(Manifest.permission.READ_MEDIA_IMAGES)
        else arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE)
        else -> null
    }

    private fun permissionStatus(a: Activity, kind: String): String {
        if (kind == "photos-add-only" && Build.VERSION.SDK_INT >= 29) return "granted"
        val names = permissionNames(kind) ?: return "restricted"
        fun granted(name: String) = a.checkSelfPermission(name) == PackageManager.PERMISSION_GRANTED
        if (kind == "notifications") {
            val enabled = a.getSystemService(NotificationManager::class.java).areNotificationsEnabled()
            if (enabled && (names.isEmpty() || names.all(::granted))) return "granted"
            if (names.isEmpty()) return "denied"
        } else if (kind == "geolocation" && granted(Manifest.permission.ACCESS_COARSE_LOCATION)) {
            return "granted"
        } else if (kind == "photos" || kind == "photos-add-only") {
            if (granted(names[0])) return "granted"
            if (Build.VERSION.SDK_INT >= 34 && granted(Manifest.permission.READ_MEDIA_VISUAL_USER_SELECTED)) return "limited"
        } else if (names.all(::granted)) return "granted"
        val asked = a.getSharedPreferences("mygo-permissions", 0).getBoolean(kind, false)
        return if (asked) "denied" else "not-determined"
    }

    fun cameraForScanner(token: Long, valid: () -> Boolean, completion: (Boolean) -> Unit) {
        ui.post {
            if (!valid()) return@post
            val a = owner.get()
            if (a == null) { completion(false); return@post }
            if (permissionStatus(a, "camera") == "granted") { completion(true); return@post }
            permissions.add(PermissionRequest(-token, "camera", completion = { if (valid()) completion(it == "granted") }, valid = valid))
            nextPermission()
        }
    }
    fun discardScannerPermission(token: Long) { ui.post { permissions.removeAll { it.id == -token && it.completion != null } } }
    private fun nextPermission() {
        if (activePermission != null) return
        val a = owner.get() ?: return
        if (a.isDestroyed || a.isFinishing) return
        while (permissions.isNotEmpty()) {
            val request = permissions.removeFirst()
            if (request.valid?.invoke() == false || (request.completion == null && !pending.contains(request.id))) continue
            activePermission = request
            request.owner = a
            a.getSharedPreferences("mygo-permissions", 0).edit().putBoolean(request.kind, true).apply()
            try {
                a.requestPermissions(permissionNames(request.kind)!!, PERMISSION_REQUEST)
            } catch (e: Exception) {
                activePermission = null
                if (request.completion != null) request.completion.invoke("denied")
                else finish(request.id, code = "failed", message = e.message ?: "Permission request failed")
                continue
            }
            return
        }
    }

    fun onRequestPermissionsResult(a: Activity, requestCode: Int): Boolean {
        if (requestCode != PERMISSION_REQUEST) return false
        val request = activePermission
        if (request?.owner !== a) return true
        activePermission = null
        if (request != null) {
            if (request.completion != null) request.completion.invoke(permissionStatus(a, request.kind))
            else finish(request.id, permissionStatus(a, request.kind))
        }
        nextPermission()
        return true
    }
}
