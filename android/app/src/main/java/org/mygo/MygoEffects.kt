package org.mygo

import android.app.Activity
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.os.Build
import android.view.HapticFeedbackConstants
import android.view.View
import android.view.WindowInsets
import android.view.WindowInsetsController

internal object MygoEffects {
    private const val BADGE_ID = 0x4d59424c
    private const val BADGE_CHANNEL = "mygo-unread"
    fun haptic(a: Activity, kind: String) {
        val effect = when (kind) {
            "selection" -> HapticFeedbackConstants.TEXT_HANDLE_MOVE
            "light" -> HapticFeedbackConstants.VIRTUAL_KEY
            "medium" -> HapticFeedbackConstants.KEYBOARD_TAP
            "heavy" -> HapticFeedbackConstants.LONG_PRESS
            "soft" -> HapticFeedbackConstants.CLOCK_TICK
            "rigid" -> HapticFeedbackConstants.CONTEXT_CLICK
            "success" -> if (Build.VERSION.SDK_INT >= 30) HapticFeedbackConstants.CONFIRM else HapticFeedbackConstants.LONG_PRESS
            "warning", "error" -> if (Build.VERSION.SDK_INT >= 30) HapticFeedbackConstants.REJECT else HapticFeedbackConstants.LONG_PRESS
            else -> throw IllegalArgumentException("Unknown haptic effect")
        }
        // System settings and hardware remain authoritative.
        a.window.decorView.performHapticFeedback(effect)
    }
    fun badge(a: Activity, id: Long, count: Int) {
        require(count >= 0)
        val manager = a.getSystemService(NotificationManager::class.java)
        if (count == 0) { manager.cancel(BADGE_ID); MygoServices.complete(id, true); return }
        if (!manager.areNotificationsEnabled()) { MygoServices.complete(id, code = "notifications-denied", message = "Notifications are disabled"); return }
        if (Build.VERSION.SDK_INT >= 26) {
            val channel = NotificationChannel(BADGE_CHANNEL, "Unread conversations", NotificationManager.IMPORTANCE_LOW).apply { setShowBadge(true); setSound(null, null) }
            manager.createNotificationChannel(channel)
        }
        val launch = a.packageManager.getLaunchIntentForPackage(a.packageName) ?: Intent(a, MygoActivity::class.java)
        val intent = PendingIntent.getActivity(a, BADGE_ID, launch, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val notification = androidx.core.app.NotificationCompat.Builder(a, BADGE_CHANNEL).setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentTitle(a.applicationInfo.loadLabel(a.packageManager)).setContentText("$count unread conversations")
            .setNumber(count).setOnlyAlertOnce(true).setSilent(true).setContentIntent(intent).setAutoCancel(true).build()
        manager.notify(BADGE_ID, notification)
        MygoServices.complete(id, true)
    }
    private var themeSource = "system"
    private fun dark(a: Activity) = when (themeSource) {
        "light" -> false
        "dark" -> true
        else -> (a.resources.configuration.uiMode and android.content.res.Configuration.UI_MODE_NIGHT_MASK) == android.content.res.Configuration.UI_MODE_NIGHT_YES
    }
    fun sourceChanged(a: Activity, source: String) {
        themeSource = source
        if (a is MygoActivity) followTheme(a)
    }
    private val barOverrides = java.util.WeakHashMap<Activity, Pair<String, Boolean>>()
    fun statusBar(a: Activity, style: String, hidden: Boolean) {
        barOverrides[a] = style to hidden
        applyStatusBar(a, style, hidden)
    }
    fun followTheme(a: Activity) {
        val (style, hidden) = barOverrides[a] ?: ("" to false)
        applyStatusBar(a, style, hidden)
        val darkIcons = !dark(a)
        if (Build.VERSION.SDK_INT >= 30) {
            a.window.insetsController?.setSystemBarsAppearance(if (darkIcons) WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS else 0, WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS)
        } else if (Build.VERSION.SDK_INT >= 26) {
            val flags = a.window.decorView.systemUiVisibility
            a.window.decorView.systemUiVisibility = if (darkIcons) flags or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR else flags and View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR.inv()
        }
    }
    private fun applyStatusBar(a: Activity, style: String, hidden: Boolean) {
        val darkIcons = if (style.isEmpty()) !dark(a) else style == "dark"
        if (Build.VERSION.SDK_INT >= 30) {
            a.window.insetsController?.apply {
                setSystemBarsAppearance(if (darkIcons) WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS else 0, WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS)
                if (hidden) { systemBarsBehavior = WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE; hide(WindowInsets.Type.statusBars()) }
                else show(WindowInsets.Type.statusBars())
            }
        } else {
            var flags = a.window.decorView.systemUiVisibility
            flags = if (darkIcons) flags or View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR else flags and View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR.inv()
            flags = if (hidden) flags or View.SYSTEM_UI_FLAG_FULLSCREEN else flags and View.SYSTEM_UI_FLAG_FULLSCREEN.inv()
            a.window.decorView.systemUiVisibility = flags
        }
    }
}
