package org.mygo

import android.app.Activity
import android.app.AlertDialog
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.pdf.PdfRenderer
import android.os.ParcelFileDescriptor
import android.webkit.CookieManager
import android.webkit.URLUtil
import android.widget.Button
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

/** Fetch into private cache before opening a document provider: the latter may
 * background the app and close the local Go proxy. No public storage access or
 * JavaScript bridge is needed. */
internal class MygoBrowserFiles(private var activity: Activity) {
    companion object {
        private const val EXPORT = 23103
        private const val LIMIT = 64L * 1024 * 1024
        private val worker = Executors.newCachedThreadPool()
    }
    private class Transfer {
        val canceled = AtomicBoolean(false)
        @Volatile var connection: HttpURLConnection? = null
    }
    private class CachedFile(val path: File) {
        private val references = AtomicInteger(1)
        fun retain() { references.incrementAndGet() }
        fun release() { if (references.decrementAndGet() == 0) path.delete() }
    }
    private var closed = false
    private var transfer: Transfer? = null
    private var file: CachedFile? = null
    private var pendingExport: CachedFile? = null
    private var dialog: AlertDialog? = null
    private var pdf: PdfPreview? = null
    private var previewName: String? = null
    private var previewPage = 0
    fun attach(a: Activity) { activity = a }
    fun detach() {
        cancelTransfer(); dialog?.dismiss(); dialog = null
        val name = previewName; val page = previewPage
        pdf?.close(); pdf = null; previewName = name; previewPage = page
    }
    fun restorePreview() {
        val cached = file ?: return
        val name = previewName ?: return
        if (!closed && pdf == null) {
            pdf = PdfPreview(cached.path, name, previewPage) { export(cached, name, "application/pdf") }
            pdf?.show()
        }
    }

    fun download(url: String, userAgent: String?, disposition: String?, mime: String?) {
        if (closed || transfer != null || pendingExport != null) return
        pdf?.close(); pdf = null; previewName = null
        file?.release(); file = null
        val job = Transfer(); transfer = job
        dialog = AlertDialog.Builder(activity).setMessage("Preparing file…").setNegativeButton("Cancel") { _, _ -> cancelTransfer() }
            .setOnCancelListener { cancelTransfer() }.show()
        worker.execute {
            var result: File? = null
            try {
                var current = URL(url)
                var responseMime = mime ?: "application/octet-stream"
                var responseDisposition = disposition
                for (redirect in 0..10) {
                    check(!job.canceled.get()) { "Download canceled" }
                    require(current.protocol == "http" || current.protocol == "https") { "Only HTTP downloads are supported" }
                    val connection = current.openConnection() as HttpURLConnection
                    job.connection = connection
                    try {
                        connection.instanceFollowRedirects = false
                        connection.connectTimeout = 15000; connection.readTimeout = 15000
                        if (!userAgent.isNullOrEmpty()) connection.setRequestProperty("User-Agent", userAgent)
                        // Recompute for each origin; never forward another host's Cookie header.
                        CookieManager.getInstance().getCookie(current.toString())?.let { connection.setRequestProperty("Cookie", it) }
                        val code = connection.responseCode
                        if (code in listOf(301, 302, 303, 307, 308)) {
                            check(redirect < 10) { "Too many download redirects" }
                            val location = connection.getHeaderField("Location") ?: error("Redirect has no destination")
                            val next = URL(current, location)
                            require(current.protocol != "https" || next.protocol == "https") { "Insecure download redirect" }
                            current = next; continue
                        }
                        check(code in 200..299) { "Download failed (HTTP $code)" }
                        require(connection.contentLengthLong <= LIMIT) { "File exceeds 64 MB download limit" }
                        responseMime = connection.contentType?.substringBefore(';') ?: responseMime
                        responseDisposition = connection.getHeaderField("Content-Disposition") ?: responseDisposition
                        val output = File.createTempFile("mygo-download-", ".tmp", activity.cacheDir); result = output
                        connection.inputStream.use { input -> output.outputStream().use { stream ->
                            val buffer = ByteArray(32768); var count = 0L
                            while (true) {
                                check(!job.canceled.get()) { "Download canceled" }
                                val size = input.read(buffer); if (size < 0) break
                                count += size; require(count <= LIMIT) { "File exceeds 64 MB download limit" }; stream.write(buffer, 0, size)
                            }
                        } }
                        break
                    } finally { connection.disconnect(); job.connection = null }
                }
                val output = result ?: error("No download response")
                val name = URLUtil.guessFileName(current.toString(), responseDisposition, responseMime)
                    .substringAfterLast('/').substringAfterLast('\\').replace(Regex("[\\p{Cntrl}]"), "_").take(120).ifBlank { "download" }
                val isPDF = responseMime.equals("application/pdf", true) || name.endsWith(".pdf", true)
                activity.runOnUiThread {
                    if (closed || transfer !== job || job.canceled.get()) output.delete()
                    else {
                        transfer = null; dialog?.dismiss(); dialog = null
                        val cached = CachedFile(output); file = cached
                        if (isPDF) { previewName = name; previewPage = 0; restorePreview() }
                        else export(cached, name, responseMime)
                    }
                }
            } catch (e: Exception) {
                result?.delete()
                activity.runOnUiThread {
                    if (transfer === job) {
                        transfer = null; dialog?.dismiss(); dialog = null
                        if (!closed && !job.canceled.get()) AlertDialog.Builder(activity).setMessage(e.message ?: "Cannot download file").setPositiveButton("OK", null).show()
                    }
                }
            }
        }
    }
    private fun cancelTransfer() {
        transfer?.let { it.canceled.set(true); it.connection?.disconnect() }; transfer = null
    }
    private fun export(source: CachedFile, name: String, mime: String) {
        if (closed || pendingExport != null) return
        pendingExport = source
        try { activity.startActivityForResult(Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
            type = mime; addCategory(Intent.CATEGORY_OPENABLE); putExtra(Intent.EXTRA_TITLE, name)
        }, EXPORT) }
        catch (e: Exception) { pendingExport = null; AlertDialog.Builder(activity).setMessage(e.message ?: "Cannot export file").setPositiveButton("OK", null).show() }
    }
    fun result(request: Int, code: Int, data: Intent?): Boolean {
        if (request != EXPORT) return false
        val source = pendingExport ?: return true
        pendingExport = null
        if (code != Activity.RESULT_OK || data?.data == null) return true
        val destination = data.data!!
        // The worker keeps a reference after Activity cleanup, with all file
        // I/O on the worker rather than Android's UI thread.
        source.retain()
        worker.execute {
            try { activity.contentResolver.openOutputStream(destination, "wt")?.use { output -> source.path.inputStream().use { it.copyTo(output) } }
                ?: error("Cannot write selected document") }
            catch (e: Exception) { activity.runOnUiThread { showExportError(e) } }
            finally { source.release() }
        }
        return true
    }
    private fun showExportError(e: Exception) {
        if (!closed) AlertDialog.Builder(activity).setMessage(e.message ?: "Cannot export file").setPositiveButton("OK", null).show()
    }
    fun close() {
        closed = true; cancelTransfer(); dialog?.dismiss(); dialog = null
        pdf?.close(); pdf = null; file?.release(); file = null; pendingExport = null
    }

    /** A single bounded bitmap is rendered on a worker, not on Android's UI thread. */
    private inner class PdfPreview(private val source: File, private val name: String, private val initialPage: Int, private val save: () -> Unit) {
        private val serial = Executors.newSingleThreadExecutor()
        private var renderer: PdfRenderer? = null
        private var descriptor: ParcelFileDescriptor? = null
        private val canceled = AtomicBoolean(false)
        private var image: ImageView? = null
        private var bitmap: Bitmap? = null
        private var sheet: AlertDialog? = null
        private var index = 0
        private var pages = 0
        private lateinit var title: TextView
        private lateinit var previous: Button
        private lateinit var next: Button
        fun show() {
            val layout = LinearLayout(activity).apply { orientation = LinearLayout.VERTICAL }
            title = TextView(activity).apply { text = "Opening PDF…"; setPadding(16, 8, 16, 8) }; layout.addView(title)
            val controls = LinearLayout(activity); layout.addView(controls)
            fun button(label: String, action: () -> Unit) = Button(activity).apply { text = label; setOnClickListener { action() }; controls.addView(this, LinearLayout.LayoutParams(0, -2, 1f)) }
            previous = button("Previous page") { render(index - 1) }
            next = button("Next page") { render(index + 1) }
            button("Export") { save() }
            image = ImageView(activity).apply { adjustViewBounds = true; setBackgroundColor(Color.WHITE) }
            layout.addView(ScrollView(activity).apply { addView(image) }, LinearLayout.LayoutParams(-1, 0, 1f))
            sheet = AlertDialog.Builder(activity).setTitle(name).setView(layout).setPositiveButton("Close", null)
                .setOnDismissListener { close() }.create()
            sheet!!.show(); sheet!!.window?.setLayout(-1, (activity.resources.displayMetrics.heightPixels * .85f).toInt())
            render(initialPage)
        }
        private fun render(pageIndex: Int) {
            if (canceled.get()) return
            previous.isEnabled = false; next.isEnabled = false
            val width = activity.resources.displayMetrics.widthPixels.coerceIn(1, 1600)
            serial.execute {
                var rendered: Bitmap? = null
                try {
                    if (canceled.get()) return@execute
                    if (renderer == null) {
                        descriptor = ParcelFileDescriptor.open(source, ParcelFileDescriptor.MODE_READ_ONLY)
                        renderer = PdfRenderer(descriptor!!)
                    }
                    val document = renderer!!; pages = document.pageCount
                    require(pageIndex in 0 until pages) { "PDF has no requested page" }
                    document.openPage(pageIndex).use { page ->
                        val ratio = page.height.toDouble() / page.width.coerceAtLeast(1)
                        val w = minOf(width, kotlin.math.sqrt(2_000_000.0 / ratio).toInt().coerceAtLeast(1))
                        val h = (w * ratio).toInt().coerceIn(1, 2_000_000 / w)
                        rendered = Bitmap.createBitmap(w, h, Bitmap.Config.ARGB_8888).apply { eraseColor(Color.WHITE) }
                        page.render(rendered!!, null, null, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY)
                    }
                    val output = rendered!!
                    activity.runOnUiThread {
                        if (canceled.get()) output.recycle()
                        else {
                            image?.setImageBitmap(output); bitmap?.recycle(); bitmap = output; index = pageIndex; previewPage = index
                            title.text = "Page ${index + 1} of $pages"; image?.contentDescription = "PDF page ${index + 1} of $pages"
                            previous.isEnabled = index > 0; next.isEnabled = index + 1 < pages
                        }
                    }
                } catch (e: Exception) {
                    rendered?.recycle()
                    activity.runOnUiThread { if (!canceled.get()) title.text = e.message ?: "Cannot preview PDF. Use Export to open it elsewhere." }
                }
            }
        }
        fun close() {
            if (!canceled.compareAndSet(false, true)) return
            previewName = null
            sheet?.dismiss(); sheet = null; image?.setImageDrawable(null); bitmap?.recycle(); bitmap = null
            serial.execute { renderer?.close(); renderer = null; descriptor?.close(); descriptor = null }
            serial.shutdown()
        }
    }
}
