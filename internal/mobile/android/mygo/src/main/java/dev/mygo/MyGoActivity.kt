package dev.mygo

import android.content.Intent
import android.content.res.Configuration
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.core.view.WindowCompat

/**
 * The app's one Activity. Go and the app's windows live in the process,
 * not the Activity: an Activity Android makes again takes them over.
 */
class MyGoActivity : ComponentActivity() {
    private var lastConfig: Configuration? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        WindowCompat.setDecorFitsSystemWindows(window, false)
        lastConfig = Configuration(resources.configuration)
        tintBars()
        HostImpl.attach(this)
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                // Back the app does not take leaves it running, as the
                // system's launcher does with apps it goes back from.
                if (!HostImpl.back()) moveTaskToBack(true)
            }
        })
        HostImpl.startGo(this, intent)
        openIntent(intent)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        openIntent(intent)
    }

    private fun openIntent(intent: Intent?) {
        intent ?: return
        // A notification tapped: once, as the Activity is made again with
        // the same Intent after Android ends the process.
        intent.getStringExtra("mygo.notification")?.let {
            intent.removeExtra("mygo.notification")
            Go.send(Event.notificationClicked(it))
        }
        val data = intent.data ?: return
        if (intent.action == Intent.ACTION_VIEW) Go.send(Event.openURLs(listOf(data.toString())))
    }

    override fun onResume() {
        super.onResume()
        Go.ask(Event.background(false))
        Go.send(Event.active(true))
    }

    override fun onPause() {
        Go.send(Event.active(false))
        super.onPause()
    }

    override fun onStop() {
        Go.ask(Event.background(true))
        super.onStop()
    }

    override fun onConfigurationChanged(c: Configuration) {
        super.onConfigurationChanged(c)
        val old = lastConfig
        lastConfig = Configuration(c)
        if (old == null) return
        val diff = old.diff(c)
        if (diff and (ActivityInfoChanges.UI_MODE or ActivityInfoChanges.FONT_SCALE) != 0) {
            tintBars()
            Go.send(Event.themeChanged())
        }
        if (diff and ActivityInfoChanges.LOCALE != 0) Go.send(Event.localeChanged())
    }

    /** Dark icons on the system's bars in light mode, light ones in dark mode. */
    private fun tintBars() {
        val dark = resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK == Configuration.UI_MODE_NIGHT_YES
        WindowCompat.getInsetsController(window, window.decorView).apply {
            isAppearanceLightStatusBars = !dark
            isAppearanceLightNavigationBars = !dark
        }
    }

    override fun onLowMemory() {
        super.onLowMemory()
        Go.send(Event.memoryWarning())
    }

    override fun onTrimMemory(level: Int) {
        super.onTrimMemory(level)
        // Running low, or next to be ended in the background.
        @Suppress("DEPRECATION")
        if (level == TRIM_MEMORY_RUNNING_LOW || level == TRIM_MEMORY_RUNNING_CRITICAL || level >= TRIM_MEMORY_MODERATE) {
            Go.send(Event.memoryWarning())
        }
    }

    override fun onDestroy() {
        HostImpl.detach(this)
        super.onDestroy()
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        HostImpl.activityResult(requestCode, resultCode, data)
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<String>, results: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, results)
        HostImpl.permissionsResult(requestCode, results)
    }
}

/** ActivityInfo's configuration change bits, as Configuration.diff gives them. */
private object ActivityInfoChanges {
    const val LOCALE = android.content.pm.ActivityInfo.CONFIG_LOCALE
    const val UI_MODE = android.content.pm.ActivityInfo.CONFIG_UI_MODE
    const val FONT_SCALE = android.content.pm.ActivityInfo.CONFIG_FONT_SCALE
}
