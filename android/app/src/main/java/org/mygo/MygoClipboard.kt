package org.mygo

import android.content.ClipData
import android.content.ClipDescription
import android.content.ClipboardManager
import android.content.Context
import android.os.Build
import android.os.PersistableBundle
import androidx.core.content.FileProvider
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.util.UUID

/** Native clipboard alternatives: plain/HTML inline; Markdown through a typed URI.
 * The extra is a small immediate representation for mygo readers. Other Android
 * apps can read the same Markdown through ContentResolver and text/markdown.
 */
internal object MygoClipboard {
    private const val MARKDOWN = "text/markdown"
    private const val EXTRA = "org.mygo.clipboard.markdown"
    private const val MAX_MARKDOWN = 1024 * 1024
    private val lock = Any()
    private fun manager(ctx: Context) = ctx.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    private fun directory(ctx: Context) = File(ctx.cacheDir, "mygo-clipboard")

    fun read(ctx: Context?): ByteArray = synchronized(lock) {
        val out = JSONObject().put("mimes", JSONArray())
        if (ctx != null) try {
            val clip = manager(ctx).primaryClip
            if (clip != null && clip.itemCount > 0) {
                val desc = clip.description
                val mimes = JSONArray()
                for (i in 0 until desc.mimeTypeCount) mimes.put(desc.getMimeType(i))
                out.put("mimes", mimes)
                val item = clip.getItemAt(0)
                item.text?.let { out.put("plain", it.toString()) }
                item.htmlText?.let { out.put("html", it) }
                val types = (0 until desc.mimeTypeCount).map { desc.getMimeType(it).substringBefore(';').trim().lowercase(java.util.Locale.ROOT) }.toSet()
                if (MARKDOWN in types || "text/x-markdown" in types) {
                    var markdown = desc.extras?.getString(EXTRA)
                    if (markdown != null && markdown.toByteArray(Charsets.UTF_8).size > MAX_MARKDOWN) markdown = null
                    // Never synchronously open a third-party provider on the Go
                    // pump. Only our own small private cache file is a fallback.
                    val uri = item.uri
                    if (markdown == null && uri?.authority == ctx.packageName + ".fileprovider" && uri.pathSegments.size == 2 && uri.pathSegments[0] == "clipboard") {
                        val name = uri.pathSegments[1]
                        if (Regex("[a-f0-9-]{36}\\.md").matches(name)) {
                            val file = File(directory(ctx), name)
                            if (file.isFile && file.length() <= MAX_MARKDOWN) markdown = file.readText(Charsets.UTF_8)
                        }
                    }
                    if (markdown == null && "text/plain" !in types && "text/html" !in types) markdown = item.text?.toString()
                    markdown?.let { out.put("markdown", it) }
                }
            }
        } catch (_: Exception) { /* Focus restrictions are an unreadable clipboard. */ }
        out.toString().toByteArray(Charsets.UTF_8)
    }

    fun write(ctx: Context?, data: ByteArray): Boolean = synchronized(lock) {
        if (ctx == null) return@synchronized false
        var file: File? = null
        try {
            val value = JSONObject(String(data, Charsets.UTF_8))
            fun text(key: String): String? = if (value.has(key) && !value.isNull(key)) value.getString(key) else null
            val html = text("html")
            // Android requires a plain fallback whenever an HTML item is set.
            val plain = text("plain") ?: html?.let { android.text.Html.fromHtml(it, android.text.Html.FROM_HTML_MODE_LEGACY).toString() }
            val markdown = text("markdown")
            if (plain == null && html == null && markdown == null) return@synchronized false
            val mimes = ArrayList<String>(3)
            if (plain != null) mimes.add("text/plain")
            if (html != null) mimes.add("text/html")
            var uri: android.net.Uri? = null
            if (markdown != null) {
                val bytes = markdown.toByteArray(Charsets.UTF_8)
                if (bytes.size > MAX_MARKDOWN) return@synchronized false
                val dir = directory(ctx)
                if (!dir.isDirectory && !dir.mkdirs()) return@synchronized false
                file = File(dir, UUID.randomUUID().toString() + ".md")
                file!!.writeBytes(bytes)
                uri = FileProvider.getUriForFile(ctx, ctx.packageName + ".fileprovider", file!!)
                mimes.add(MARKDOWN)
            }
            val description = ClipDescription("mygo", mimes.toTypedArray())
            if (markdown != null && markdown.toByteArray(Charsets.UTF_8).size <= 64 * 1024) {
                description.extras = PersistableBundle().apply { putString(EXTRA, markdown) }
            }
            manager(ctx).setPrimaryClip(ClipData(description, ClipData.Item(plain, html, null, uri)))
            cleanup(ctx, file)
            true
        } catch (_: Exception) {
            file?.delete()
            false
        }
    }

    fun clear(ctx: Context?): Boolean = synchronized(lock) {
        if (ctx == null) return@synchronized false
        try {
            if (Build.VERSION.SDK_INT >= 28) manager(ctx).clearPrimaryClip()
            else manager(ctx).setPrimaryClip(ClipData.newPlainText("", ""))
            cleanup(ctx, null)
            true
        } catch (_: RuntimeException) { false }
    }

    private fun cleanup(ctx: Context, keep: File?) {
        // Representations have already been copied; old owners were replaced.
        // The current URI survives Activity/process restarts, with at most one
        // cache file retained. FileProvider remains non-exported and grants read
        // access only to the clipboard recipient, never the whole directory.
        try { directory(ctx).listFiles()?.forEach { if (it != keep && it.isFile) it.delete() } }
        catch (_: Exception) { /* Cleanup must not invalidate a successful write. */ }
    }
}
