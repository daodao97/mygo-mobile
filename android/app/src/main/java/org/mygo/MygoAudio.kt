package org.mygo

import android.Manifest
import android.app.Activity
import android.content.pm.PackageManager
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference

/** 16 kHz mono signed PCM. No recorder operation waits on Android's UI thread. */
internal object MygoAudio {
    private class Capture(val token: Long) {
        val mode = AtomicInteger(0) // 0 recording, 1 finish, 2 discard
        val lock = Any()
        var recorder: AudioRecord? = null
    }
    private val reader = Executors.newSingleThreadExecutor()
    private val control = Executors.newSingleThreadExecutor()
    private val active = AtomicReference<Capture?>(null)

    fun start(a: Activity, id: Long, token: Long) {
        if (active.get() != null) { MygoServices.complete(id, code = "busy", message = "Audio capture is already active"); return }
        if (!MygoServices.isForeground() || a.checkSelfPermission(Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) {
            MygoServices.complete(id, code = "denied", message = "Microphone permission or foreground access is missing"); return
        }
        val capture = Capture(token)
        active.set(capture)
        MygoServices.onCancel(id) { stop(token, true) }
        reader.execute {
            var failure: String? = null
            try {
                val minimum = AudioRecord.getMinBufferSize(16000, AudioFormat.CHANNEL_IN_MONO, AudioFormat.ENCODING_PCM_16BIT)
                check(minimum > 0) { "Microphone format unavailable" }
                val recorder = AudioRecord.Builder().setAudioSource(MediaRecorder.AudioSource.VOICE_RECOGNITION)
                    .setAudioFormat(AudioFormat.Builder().setSampleRate(16000).setChannelMask(AudioFormat.CHANNEL_IN_MONO).setEncoding(AudioFormat.ENCODING_PCM_16BIT).build())
                    .setBufferSizeInBytes(maxOf(minimum, 6400)).build()
                synchronized(capture.lock) { capture.recorder = recorder }
                check(recorder.state == AudioRecord.STATE_INITIALIZED) { "Cannot initialize microphone" }
                if (capture.mode.get() == 0) recorder.startRecording()
                MygoServices.post { MygoServices.complete(id, true) }
                val buffer = ByteArray(640) // 20 ms, always little-endian PCM16
                while (capture.mode.get() == 0) {
                    val n = recorder.read(buffer, 0, buffer.size, AudioRecord.READ_BLOCKING)
                    if (n > 0 && capture.mode.get() != 2) Native.onAudio(token, buffer.copyOf(n), null, false)
                    else if (n < 0 && capture.mode.get() == 0) throw IllegalStateException("Microphone read failed: $n")
                }
            } catch (e: Exception) {
                if (capture.mode.get() == 0) failure = e.message ?: "Audio capture failed"
                MygoServices.post { MygoServices.complete(id, code = "failed", message = failure ?: "Capture canceled") }
            } finally {
                synchronized(capture.lock) {
                    capture.recorder?.let { recorder -> try { recorder.stop() } catch (_: Exception) {} ; recorder.release() }
                    capture.recorder = null
                }
                active.compareAndSet(capture, null)
                Native.onAudio(token, null, failure?.toByteArray(Charsets.UTF_8), true)
            }
        }
    }

    fun stop(token: Long, cancel: Boolean) {
        val capture = active.get() ?: return
        if (capture.token != token) return
        if (cancel) capture.mode.set(2) else capture.mode.compareAndSet(0, 1)
        // Stop from a control worker to release a blocking read even if the
        // input device disappears. release belongs to the reader's finally.
        control.execute { synchronized(capture.lock) { try { capture.recorder?.stop() } catch (_: Exception) {} } }
    }
    fun background() { active.get()?.let { stop(it.token, true) } }
}
