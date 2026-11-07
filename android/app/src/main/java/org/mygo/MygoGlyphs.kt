package org.mygo

import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.fonts.Font
import android.os.Build
import java.io.File
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.ConcurrentHashMap
import kotlin.math.ceil
import kotlin.math.floor
import org.json.JSONObject

/** Canvas/Skia rasterizes COLR/CBDT glyphs without involving the UI thread. */
internal object MygoGlyphs {
    private val fonts = ConcurrentHashMap<String, Font>()
    fun raster(bytes: ByteArray): ByteArray? {
        if (Build.VERSION.SDK_INT < 31) return null
        return try {
            val options = JSONObject(String(bytes, Charsets.UTF_8))
            val size = options.getDouble("size").toFloat()
            if (!size.isFinite() || size <= 0f || size > 2048f) return null
            val path = options.getString("path"); val index = options.getInt("index")
            val key = "$path:$index"
            val font = fonts[key] ?: Font.Builder(File(path)).setTtcIndex(index).build().also {
                if (fonts.size >= 64) fonts.clear(); fonts[key] = it
            }
            val glyph = options.getInt("glyph"); val dx = options.getDouble("dx").toFloat()
            val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply { textSize = size; color = Color.WHITE }
            val bounds = RectF(); font.getGlyphBounds(glyph, paint, bounds)
            val left = floor(bounds.left + dx).toInt() - 1; val top = floor(bounds.top).toInt() - 1
            val width = ceil(bounds.right + dx).toInt() + 1 - left
            val height = ceil(bounds.bottom).toInt() + 1 - top
            if (width <= 0 || height <= 0 || width > 2048 || height > 2048) return null
            val bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
            try {
                Canvas(bitmap).drawGlyphs(intArrayOf(glyph), 0, floatArrayOf(dx - left, -top.toFloat()), 0, 1, font, paint)
                val pixels = IntArray(width * height); bitmap.getPixels(pixels, 0, width, 0, 0, width, height)
                val result = ByteBuffer.allocate(16 + pixels.size * 4).order(ByteOrder.LITTLE_ENDIAN)
                result.putInt(width).putInt(height).putInt(left).putInt(top)
                // getPixels returns non-premultiplied ARGB; the color atlas
                // consumes premultiplied RGBA, including antialiased edges.
                for (pixel in pixels) {
                    val alpha = pixel ushr 24
                    result.put((((pixel ushr 16) and 255) * alpha / 255).toByte())
                    result.put((((pixel ushr 8) and 255) * alpha / 255).toByte())
                    result.put(((pixel and 255) * alpha / 255).toByte()).put(alpha.toByte())
                }
                result.array()
            } finally { bitmap.recycle() }
        } catch (e: Exception) { android.util.Log.w("mygo", "Color glyph rasterization failed: ${e.javaClass.simpleName}"); null }
    }
}
