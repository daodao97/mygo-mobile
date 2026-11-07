package org.mygo

import android.app.Activity
import android.app.AlertDialog
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.DocumentsContract
import android.provider.MediaStore
import android.provider.OpenableColumns
import android.webkit.MimeTypeMap
import android.widget.CheckBox
import java.io.File
import java.util.UUID
import java.util.concurrent.Executors
import org.json.JSONArray
import org.json.JSONObject

/** System-owned selection; imported files are copied into private cache. */
internal object MygoPickers {
    private const val REQUEST_BASE = 25000
    private var nextRequest = REQUEST_BASE
    private val pendingCodes = mutableSetOf<Int>()
    private const val FILE_LIMIT = 64L * 1024 * 1024
    private const val BATCH_LIMIT = 256L * 1024 * 1024
    private val worker = Executors.newSingleThreadExecutor()
    private data class Selection(val id: Long, val kind: String, val options: JSONObject, var owner: Activity, val requestCode: Int)
    private var active: Selection? = null
    private val messages = mutableMapOf<Long, Pair<Activity, AlertDialog>>()

    fun start(a: Activity, id: Long, kind: String, options: JSONObject) {
        if (active != null) { MygoServices.complete(id, code = "busy", message = "Another picker is open"); return }
        if (kind == "open" && options.optBoolean("OpenDirectories")) {
            MygoServices.complete(id, code = "unsupported", message = "Directory leases are not filesystem paths"); return
        }
        val intent = when (kind) {
            "photos" -> if (Build.VERSION.SDK_INT >= 33) Intent(MediaStore.ACTION_PICK_IMAGES).apply {
                type = "image/*"
                if (options.optBoolean("Multiple")) putExtra(MediaStore.EXTRA_PICK_IMAGES_MAX, minOf(16, MediaStore.getPickImagesMaxLimit()))
            } else Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                type = "image/*"; addCategory(Intent.CATEGORY_OPENABLE); putExtra(Intent.EXTRA_ALLOW_MULTIPLE, options.optBoolean("Multiple"))
            }
            "open" -> Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                type = "*/*"; addCategory(Intent.CATEGORY_OPENABLE)
                putExtra(Intent.EXTRA_ALLOW_MULTIPLE, options.optBoolean("Multiple"))
                val types = mutableSetOf<String>()
                val filters = options.optJSONArray("Filters")
                for (i in 0 until (filters?.length() ?: 0)) {
                    val filter = filters!!.getJSONObject(i)
                    val mimes = filter.optJSONArray("MIMETypes")
                    for (j in 0 until (mimes?.length() ?: 0)) types.add(mimes!!.getString(j))
                    val extensions = filter.optJSONArray("Extensions")
                    for (j in 0 until (extensions?.length() ?: 0)) MimeTypeMap.getSingleton().getMimeTypeFromExtension(extensions!!.getString(j).trimStart('.').lowercase())?.let { types.add(it) }
                }
                if (types.isNotEmpty()) putExtra(Intent.EXTRA_MIME_TYPES, types.toTypedArray())
            }
            "export" -> {
                val files = options.getJSONArray("Files")
                if (files.length() == 1) Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
                    addCategory(Intent.CATEGORY_OPENABLE)
                    val file = File(files.getString(0)); type = mime(file.name)
                    putExtra(Intent.EXTRA_TITLE, file.name)
                } else Intent(Intent.ACTION_OPEN_DOCUMENT_TREE)
            }
            else -> throw IllegalArgumentException("Unknown picker")
        }
        val code = (0 until 1024).map { REQUEST_BASE + ((nextRequest - REQUEST_BASE + it) % 1024) }.firstOrNull { it !in pendingCodes }
        if (code == null) { MygoServices.complete(id, code = "busy", message = "Too many pending picker results"); return }
        nextRequest = REQUEST_BASE + ((code - REQUEST_BASE + 1) % 1024)
        pendingCodes.add(code)
        active = Selection(id, kind, options, a, code)
        MygoServices.onCancel(id) { active?.takeIf { it.id == id }?.let { selection -> active = null; try { selection.owner.finishActivity(selection.requestCode) } catch (_: Exception) {} } }
        try { a.startActivityForResult(intent, code) }
        catch (e: Exception) { pendingCodes.remove(code); active = null; MygoServices.complete(id, code = "failed", message = e.message ?: "Cannot open picker") }
    }

    fun result(a: Activity, requestCode: Int, resultCode: Int, data: Intent?): Boolean {
        if (requestCode !in pendingCodes) return false
        val selection = active
        if (selection?.requestCode == requestCode && selection.owner !== a) return true
        pendingCodes.remove(requestCode)
        if (selection == null || selection.requestCode != requestCode) return true
        active = null
        if (resultCode != Activity.RESULT_OK || data == null) {
            MygoServices.complete(selection.id, if (selection.kind == "export") false else JSONArray()); return true
        }
        val uris = mutableListOf<Uri>()
        data.clipData?.let { clip -> for (i in 0 until clip.itemCount) uris.add(clip.getItemAt(i).uri) }
        if (uris.isEmpty()) data.data?.let { uris.add(it) }
        if (uris.isEmpty() || uris.size > 16) {
            MygoServices.complete(selection.id, code = "failed", message = "Invalid selection count"); return true
        }
        val context = a.applicationContext
        worker.execute {
            var imported: File? = null
            try {
                if (selection.kind == "export") {
                    val files = selection.options.getJSONArray("Files")
                    val target = uris.first()
                    for (i in 0 until files.length()) {
                        val file = File(files.getString(i))
                        require(file.isFile) { "Export source is not a file" }
                        val uri = if (files.length() == 1) target else DocumentsContract.createDocument(context.contentResolver,
                            DocumentsContract.buildDocumentUriUsingTree(target, DocumentsContract.getTreeDocumentId(target)), mime(file.name), file.name)
                            ?: throw java.io.IOException("Cannot create export document")
                        context.contentResolver.openOutputStream(uri, "wt")?.use { output -> file.inputStream().use { it.copyTo(output) } }
                            ?: throw java.io.IOException("Cannot write export document")
                    }
                    MygoServices.post { MygoServices.complete(selection.id, true) }
                } else {
                    val dir = File(context.cacheDir, "imports/" + UUID.randomUUID().toString())
                    check(dir.mkdirs()) { "Cannot create import directory" }; imported = dir
                    val paths = JSONArray(); var total = 0L
                    for ((index, uri) in uris.withIndex()) {
                        var name = "attachment"
                        context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
                            if (cursor.moveToFirst() && !cursor.isNull(0)) name = cursor.getString(0)
                        }
                        name = name.substringAfterLast('/').substringAfterLast('\\').replace(Regex("[\\p{Cntrl}]"), "_").take(120)
                        if (name.isBlank() || name == "." || name == "..") name = "attachment"
                        val file = File(dir, "$index-$name")
                        var size = 0L
                        context.contentResolver.openInputStream(uri)?.use { input -> file.outputStream().use { output ->
                            val buffer = ByteArray(32768)
                            while (true) { val n = input.read(buffer); if (n < 0) break; size += n; total += n
                                require(size <= FILE_LIMIT && total <= BATCH_LIMIT) { "Selected files exceed import limit" }; output.write(buffer, 0, n) }
                        } } ?: throw java.io.IOException("Cannot read selected document")
                        paths.put(file.absolutePath)
                    }
                    MygoServices.post {
                        if (MygoServices.isPending(selection.id)) MygoServices.complete(selection.id, paths)
                        else dir.deleteRecursively()
                    }
                }
            } catch (e: Exception) {
                imported?.deleteRecursively()
                MygoServices.post { MygoServices.complete(selection.id, code = "failed", message = e.message ?: "Document operation failed") }
            }
        }
        return true
    }

    fun message(a: Activity, id: Long, options: JSONObject) {
        val content = android.widget.LinearLayout(a).apply { orientation = android.widget.LinearLayout.VERTICAL
            val pad = (24 * a.resources.displayMetrics.density).toInt(); setPadding(pad, pad / 2, pad, pad / 2) }
        val detail = options.optString("Detail")
        val text = options.optString("Message") + if (detail.isEmpty()) "" else "\n\n$detail"
        if (text.isNotEmpty()) content.addView(android.widget.TextView(a).apply { this.text = text; textSize = 16f })
        val checkbox = CheckBox(a).apply { this.text = options.optString("CheckboxLabel"); isChecked = options.optBoolean("CheckboxChecked") }
        if (checkbox.text.isNotEmpty()) content.addView(checkbox)
        val scroll = android.widget.ScrollView(a).apply { addView(content) }
        val dialog = AlertDialog.Builder(a).setTitle(options.optString("Title")).setView(scroll).create()
        val buttons = options.getJSONArray("Buttons")
        for (index in 0 until buttons.length()) content.addView(android.widget.Button(a).apply {
            this.text = buttons.getString(index)
            setOnClickListener {
                MygoServices.complete(id, JSONObject().put("Response", index).put("CheckboxChecked", checkbox.isChecked)); dialog.dismiss()
            }
        })
        messages[id] = a to dialog
        dialog.setOnDismissListener { messages.remove(id) }
        dialog.setOnCancelListener { MygoServices.complete(id, JSONObject().put("Response", options.optInt("CancelID", -1)).put("CheckboxChecked", checkbox.isChecked)) }
        MygoServices.onCancel(id) { dialog.dismiss() }
        dialog.show()
    }

    fun created(a: Activity) {
        active?.let { selection ->
            val previous = selection.owner
            if (previous.isChangingConfigurations && previous.javaClass == a.javaClass && previous.taskId == a.taskId) selection.owner = a
        }
    }
    fun destroyed(a: Activity) {
        active?.takeIf { it.owner === a && !a.isChangingConfigurations }?.let { selection ->
            active = null
            pendingCodes.remove(selection.requestCode)
            try { a.finishActivity(selection.requestCode) } catch (_: Exception) {}
            MygoServices.complete(selection.id, code = "canceled", message = "Picker Activity was destroyed")
        }
        for ((id, owned) in messages.toMap()) if (owned.first === a) {
            owned.second.cancel()
            // Dialog.cancel normally completes through OnCancel. Ensure a
            // detached or already dismissed window cannot retain a waiter.
            if (MygoServices.isPending(id)) MygoServices.complete(id, code = "canceled", message = "Dialog Activity was destroyed")
            messages.remove(id)
        }
    }

    private fun mime(name: String) = MimeTypeMap.getSingleton().getMimeTypeFromExtension(name.substringAfterLast('.', "").lowercase()) ?: "application/octet-stream"
}
