package org.mygo

import android.net.Uri
import androidx.core.content.FileProvider

/** Explicit Markdown MIME type, including Android releases without an md mapping. */
class MygoFileProvider : FileProvider() {
    override fun getType(uri: Uri): String? =
        if (uri.pathSegments.firstOrNull() == "clipboard") "text/markdown" else super.getType(uri)
}
