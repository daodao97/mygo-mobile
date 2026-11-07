package org.mygo

import android.app.Activity
import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkRequest
import android.provider.Settings
import org.json.JSONArray
import org.json.JSONObject

internal object MygoEnvironment {
 init{System.loadLibrary("mygo")}
 private var context:Context?=null
 private var preferenceObserver:android.database.ContentObserver?=null
 private var networkCallback:ConnectivityManager.NetworkCallback?=null
 fun install(a:Activity){
  context=a.applicationContext
  if(preferenceObserver==null){
   val observer=object:android.database.ContentObserver(android.os.Handler(android.os.Looper.getMainLooper())){
    override fun onChange(selfChange:Boolean){report()}
   }
   a.applicationContext.contentResolver.registerContentObserver(Settings.Global.getUriFor(Settings.Global.ANIMATOR_DURATION_SCALE),false,observer)
   preferenceObserver=observer
  }
  if(networkCallback==null){
   val cm=a.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
   val cb=object:ConnectivityManager.NetworkCallback(){override fun onAvailable(n:Network){report()};override fun onLost(n:Network){report()};override fun onLinkPropertiesChanged(n:Network,p:android.net.LinkProperties){report()}}
   try{cm.registerDefaultNetworkCallback(cb);networkCallback=cb}catch(_:RuntimeException){}
  }
  report()
 }
 fun report(){
  val a=context?:return
  val list=JSONArray()
  try{val all=java.net.NetworkInterface.getNetworkInterfaces();while(all.hasMoreElements()){
   val ni=all.nextElement();val addresses=JSONArray();for(addr in ni.interfaceAddresses){val ip=addr.address?.hostAddress?:continue;addresses.put(ip.substringBefore('%')+"/"+addr.networkPrefixLength)}
   list.put(JSONObject().put("name",ni.name).put("index",ni.index).put("mtu",ni.mtu.coerceAtLeast(1280)).put("up",ni.isUp).put("loopback",ni.isLoopback).put("addresses",addresses))
  }}catch(_:Exception){}
  val reduce=Settings.Global.getFloat(a.contentResolver,Settings.Global.ANIMATOR_DURATION_SCALE,1f)==0f
  val data=JSONObject().put("files",a.filesDir.absolutePath).put("cache",a.cacheDir.absolutePath).put("textScale",a.resources.configuration.fontScale.toDouble()).put("reduceMotion",reduce).put("interfaces",list)
  Native.onEnvironment(data.toString().toByteArray(Charsets.UTF_8))
 }
}
