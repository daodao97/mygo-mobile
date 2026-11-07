package org.mygo

import android.graphics.Canvas
import android.graphics.ColorFilter
import android.graphics.Paint
import android.graphics.PixelFormat
import android.graphics.Rect
import android.graphics.RectF
import android.graphics.drawable.Drawable
import android.util.TypedValue
import android.view.ActionMode
import android.view.Gravity
import android.view.Menu
import android.view.MenuItem
import android.view.MotionEvent
import android.view.View
import android.widget.PopupWindow
import org.json.JSONObject
import java.lang.ref.WeakReference
import kotlin.math.roundToInt

/** Go owns text, ranges and layout. Android owns only handles and the floating
 * menu. Every action is asynchronous and carries the focused field ID. */
object MygoSelection {
    external fun nativeRefresh()
    external fun nativeAction(id: Long, action: Int)
    external fun nativeDrag(id: Long, edge: Int, x: Float, y: Float)
    private var owner = WeakReference<View>(null)
    private var field = 0L
    private var readOnly = false
    private var mode: ActionMode? = null
    private var ending = false
        private var commands = emptyList<Int>()
    private var bounds = RectF()
    private var startVisible = false
    private var endVisible = false
    private var start = RectF()
    private var end = RectF()
    private var rects = emptyList<RectF>()
    private var overlay: SelectionPaint? = null
    private var handles = emptyList<PopupWindow>()
    private val labels = mapOf(1 to android.R.string.copy, 2 to android.R.string.cut,
        3 to android.R.string.paste, 4 to android.R.string.selectAll,
        5 to R.string.mygo_undo, 6 to R.string.mygo_redo)
    private fun rect(j: JSONObject?): RectF {
        if (j == null) return RectF()
        val x = j.optDouble("X").toFloat(); val y = j.optDouble("Y").toFloat()
        return RectF(x, y, x + j.optDouble("W").toFloat(), y + j.optDouble("H").toFloat())
    }
    fun update(view: View, bytes: ByteArray) {
        if (!view.hasWindowFocus()) { close(view); return }
        val j = try { JSONObject(String(bytes, Charsets.UTF_8)) } catch (_: Exception) { return }
        val id = j.optLong("ID")
        if (!j.optBoolean("Active")) {
            if (owner.get() === view) close(view)
            // Keep the current field for an IME paste with a collapsed caret.
            owner = WeakReference(view); field = id
            return
        }
        if (owner.get() !== view || field != id) { owner.get()?.let { close(it) }; owner = WeakReference(view); field = id }
        readOnly = j.optBoolean("ReadOnly")
        start = rect(j.optJSONObject("Start")); end = rect(j.optJSONObject("End"))
        bounds = rect(j.optJSONObject("Bounds"))
        startVisible=j.optBoolean("StartVisible");endVisible=j.optBoolean("EndVisible")
        val rs = j.optJSONArray("Rects")
        rects = if (rs == null) emptyList() else (0 until rs.length()).map { rect(rs.optJSONObject(it)) }
        val cs = j.optJSONArray("Commands")
        val next = if (cs == null) emptyList() else (0 until cs.length()).map { cs.optInt(it) }
        val changed = next != commands; commands = next
        if (overlay == null && readOnly) {
            overlay = SelectionPaint(view); view.overlay.add(overlay!!)
        }
        overlay?.setBounds(0, 0, view.width, view.height); overlay?.invalidateSelf()
        if (mode == null) {
            mode = view.startActionMode(object : ActionMode.Callback2() {
                override fun onCreateActionMode(m: ActionMode, menu: Menu): Boolean { fill(menu, view); return true }
                override fun onPrepareActionMode(m: ActionMode, menu: Menu): Boolean { menu.clear(); fill(menu, view); return true }
                override fun onActionItemClicked(m: ActionMode, item: MenuItem): Boolean {
                    if (owner.get() !== view || item.itemId !in commands) return false
                    nativeAction(field, item.itemId); return true
                }
                override fun onDestroyActionMode(m: ActionMode) {
                    if (mode === m) mode = null
                    if (!ending && owner.get() === view) { val old = field; close(view); nativeAction(old, 0) }
                }
                override fun onGetContentRect(m: ActionMode, v: View, out: Rect) {
                    val d = v.resources.displayMetrics.density
                    val r = rects.firstOrNull() ?: bounds
                    out.set((r.left*d).roundToInt(), (r.top*d).roundToInt(), (r.right*d).roundToInt(), (r.bottom*d).roundToInt())
                }
            }, ActionMode.TYPE_FLOATING)
        } else { if (changed) mode?.invalidate(); mode?.invalidateContentRect() }
        if (handles.isEmpty()) handles = listOf(makeHandle(view,0), makeHandle(view,1))
        positionHandles(view)
    }
    private fun fill(menu: Menu, view: View) {
        for (id in commands) labels[id]?.let { menu.add(0,id,id,view.context.getString(it)).setShowAsAction(MenuItem.SHOW_AS_ACTION_IF_ROOM) }
    }
    private fun makeHandle(view: View, edge: Int): PopupWindow {
        val size = (44*view.resources.displayMetrics.density).roundToInt()
        val attr = if (edge==0) android.R.attr.textSelectHandleLeft else android.R.attr.textSelectHandleRight
        val a = view.context.obtainStyledAttributes(intArrayOf(attr))
        val drawable = a.getDrawable(0); a.recycle()
        val handle = object : View(view.context) {
            override fun onDraw(c: Canvas) {
                val w = minOf(width, drawable?.intrinsicWidth ?: width)
                val h = minOf(height, drawable?.intrinsicHeight ?: height)
                // Tip stays at the horizontal centre of the touch target.
                drawable?.setBounds(if(edge==0) width/2-w else width/2,0,if(edge==0) width/2 else width/2+w,h)
                drawable?.draw(c)
            }
        }
        val popup = PopupWindow(handle,size,size,false).apply {
            isTouchable=true; isOutsideTouchable=false; isClippingEnabled=true
            inputMethodMode=PopupWindow.INPUT_METHOD_NOT_NEEDED
        }
        val boundID = field
        var offsetX=0f; var offsetY=0f
        handle.setOnTouchListener { _, event ->
            if (owner.get() !== view || field!=boundID) return@setOnTouchListener false
            val d=view.resources.displayMetrics.density
            when(event.actionMasked) {
                MotionEvent.ACTION_DOWN -> {
                    val loc=IntArray(2);view.getLocationOnScreen(loc)
                    val r=if(edge==0) start else end
                    offsetX=event.rawX-(loc[0]+r.left*d)
                    offsetY=event.rawY-(loc[1]+(r.top+r.height()/2)*d)
                    mode?.hide(3000)
                }
                MotionEvent.ACTION_MOVE, MotionEvent.ACTION_UP -> {
                    val loc=IntArray(2);view.getLocationOnScreen(loc)
                    nativeDrag(boundID,edge,(event.rawX-loc[0]-offsetX)/d,(event.rawY-loc[1]-offsetY)/d)
                    if(event.actionMasked==MotionEvent.ACTION_UP) {mode?.hide(0);mode?.invalidateContentRect()}
                }
                MotionEvent.ACTION_CANCEL -> {mode?.hide(0)}
            }
            true
        }
        return popup
    }
    private fun positionHandles(view: View) {
        val loc=IntArray(2);view.getLocationOnScreen(loc)
        val d=view.resources.displayMetrics.density
        for ((i,popup) in handles.withIndex()) {
            val r=if(i==0) start else end
            if (!(if(i==0) startVisible else endVisible) || r.height()<=0 || r.bottom*d<0 || r.top*d>view.height) {popup.dismiss();continue}
            val x=loc[0]+(r.left*d).roundToInt()-popup.width/2
            val y=loc[1]+(r.bottom*d).roundToInt()
            if(popup.isShowing) popup.update(x,y,-1,-1) else popup.showAtLocation(view,Gravity.TOP or Gravity.LEFT,x,y)
        }
    }
    fun contextAction(view: View, id: Int): Boolean {
        if(owner.get() !== view || field==0L) return false
        val action=when(id) {android.R.id.copy->1;android.R.id.cut->2;android.R.id.paste->3;android.R.id.selectAll->4;android.R.id.undo->5;android.R.id.redo->6;else->return false}
        nativeAction(field,action);return true
    }
    fun pause(view: View) {
        if (owner.get() !== view) return
        val id = field; val dismiss = readOnly
        close(view)
        if (dismiss) nativeAction(id, 0)
    }
    fun close(view: View) {
        if(owner.get() !== view) return
        ending=true
        val old=mode;mode=null;old?.finish()
        handles.forEach {it.dismiss()};handles=emptyList()
        overlay?.let {view.overlay.remove(it)};overlay=null
        owner.clear();field=0;commands=emptyList();rects=emptyList()
        ending=false
    }
    private class SelectionPaint(private val view: View): Drawable() {
        private val paint=Paint().apply {
            val value=TypedValue();view.context.theme.resolveAttribute(android.R.attr.colorAccent,value,true)
            color=(value.data and 0x00ffffff) or 0x50000000
        }
        override fun draw(c: Canvas) {
            val d=view.resources.displayMetrics.density
            for(r in rects) c.drawRect(r.left*d,r.top*d,r.right*d,r.bottom*d,paint)
        }
        override fun setAlpha(alpha: Int) {paint.alpha=alpha}
        override fun setColorFilter(filter: ColorFilter?) {paint.colorFilter=filter}
        override fun getOpacity(): Int = PixelFormat.TRANSLUCENT
    }
}
