package org.mygo

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.os.Build
import android.os.Bundle
import android.view.ViewGroup
import android.webkit.WebView
import android.webkit.WebViewClient
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import java.lang.ref.WeakReference

/** A native browser Activity; no JavaScript-to-native interface is exposed. */
class MygoBrowserActivity : Activity() {
    companion object {
        private var token = 0L
        private var current = WeakReference<MygoBrowserActivity>(null)
        fun open(a: Activity, id: Long, url: String) {
            if (token != 0L) { MygoServices.complete(id, code = "busy", message = "A browser is already open"); return }
            token = id
            MygoServices.onCancel(id) { if (token == id) { token = 0; current.get()?.finish() } }
            try { a.startActivity(Intent(a, MygoBrowserActivity::class.java).putExtra("id", id).putExtra("url", url)) }
            catch (e: Exception) { token = 0; MygoServices.complete(id, code = "failed", message = e.message ?: "Cannot open browser") }
        }
    }
    private lateinit var web: WebView
    private lateinit var address: TextView
    private lateinit var back: Button
    private lateinit var forward: Button
    private var requestId = 0L
    private lateinit var files: MygoBrowserFiles

    override fun onCreate(state: Bundle?) {
        super.onCreate(state)
        requestId = intent.getLongExtra("id", 0)
        if (requestId == 0L || token != requestId) { finish(); return }
        current = WeakReference(this)
        val layout = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL; setBackgroundColor(Color.WHITE) }
        val toolbar = LinearLayout(this)
        fun button(label: String, action: () -> Unit): Button = Button(this).apply { text = label; contentDescription = label; setOnClickListener { action() }; toolbar.addView(this, LinearLayout.LayoutParams(0, 56.dp(), 1f)) }
        back = button("Back") { if (web.canGoBack()) web.goBack() }
        forward = button("Forward") { if (web.canGoForward()) web.goForward() }
        button("Reload") { web.reload() }
        button("Close") { finish() }
        layout.addView(toolbar)
        address = TextView(this).apply { textSize = 12f; setPadding(12.dp(), 4.dp(), 12.dp(), 4.dp()); maxLines = 2 }
        layout.addView(address)
        if ((applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0) WebView.setWebContentsDebuggingEnabled(true)
        files = (lastNonConfigurationInstance as? MygoBrowserFiles)?.apply { attach(this@MygoBrowserActivity) } ?: MygoBrowserFiles(this)
        web = WebView(this)
        web.settings.apply {
            javaScriptEnabled = true; domStorageEnabled = true
            allowFileAccess = false; allowContentAccess = false
            mixedContentMode = WebSettings.MIXED_CONTENT_NEVER_ALLOW
            setSupportZoom(true); builtInZoomControls = true; displayZoomControls = false
        }
        web.setDownloadListener { url, userAgent, disposition, mime, _ -> files.download(url, userAgent, disposition, mime) }
        web.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                val scheme = request.url.scheme?.lowercase()
                return scheme != "http" && scheme != "https"
            }
            override fun onPageFinished(view: WebView, url: String) { address.text = url; back.isEnabled = view.canGoBack(); forward.isEnabled = view.canGoForward() }
        }
        web.webChromeClient = object : WebChromeClient() {
            override fun onReceivedTitle(view: WebView, title: String?) { this@MygoBrowserActivity.title = title ?: "Miya" }
        }
        layout.addView(web, LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f))
        setContentView(layout)
        if (Build.VERSION.SDK_INT >= 35) window.decorView.setOnApplyWindowInsetsListener { v, insets ->
            val bars = insets.getInsets(android.view.WindowInsets.Type.systemBars() or android.view.WindowInsets.Type.displayCutout())
            v.setPadding(bars.left, bars.top, bars.right, bars.bottom); insets
        }
        if (Build.VERSION.SDK_INT >= 33) onBackInvokedDispatcher.registerOnBackInvokedCallback(android.window.OnBackInvokedDispatcher.PRIORITY_DEFAULT) { goBackOrClose() }
        if (state == null || web.restoreState(state) == null) web.loadUrl(intent.getStringExtra("url")!!)
    }
    private fun Int.dp() = (this * resources.displayMetrics.density).toInt()
    private fun goBackOrClose() { if (web.canGoBack()) web.goBack() else finish() }
    @Deprecated("System callback") override fun onBackPressed() = goBackOrClose()
    override fun onSaveInstanceState(state: Bundle) { web.saveState(state); super.onSaveInstanceState(state) }
    @Deprecated("System callback") override fun onActivityResult(request: Int, result: Int, data: Intent?) {
        if (::files.isInitialized && files.result(request, result, data)) return
        if (MygoPickers.result(this, request, result, data)) return
        super.onActivityResult(request, result, data)
    }
    override fun onRequestPermissionsResult(request: Int, permissions: Array<out String>, results: IntArray) {
        if (MygoServices.onRequestPermissionsResult(this, request)) return
        super.onRequestPermissionsResult(request, permissions, results)
    }
    override fun onUserLeaveHint() { super.onUserLeaveHint(); MygoServices.onUserLeave() }
    override fun onResume() { super.onResume(); if (::files.isInitialized) files.restorePreview() }
    @Deprecated("System retention callback") override fun onRetainNonConfigurationInstance(): Any? = if (::files.isInitialized) files else null
    override fun onDestroy() {
        if (::files.isInitialized) { if (isChangingConfigurations) files.detach() else files.close() }
        if (::web.isInitialized) { web.stopLoading(); web.destroy() }
        if (!isChangingConfigurations && token == requestId) { token = 0; MygoServices.complete(requestId, "") }
        if (current.get() === this) current = WeakReference(null)
        super.onDestroy()
    }
}
