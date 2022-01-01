# diy-baby-pi
Do it yourself baby monitoring with raspberry pi!

## About

I built my own baby monitoring system with an old raspberry pi 3 I had laying around. Included in this repository are the steps
I took and software I wrote from fresh raspberry pi install to working solution. WebRTC is used to stream the video and audio for very low latency.
The steps included will be specific to the raspberry pi and hardware that I happened to use but should be able to be expanded
for use with any computer, camera, microphone, or tripod. For the purposes of this project the computer will always be referred
to as pi or raspberry pi for simplicity.

This is running in production and I am now going back re-tracing my steps and putting those steps here to recreate. The first so far has
been working so well that I am writing down these steps as I setup a second device for myself and others.

## Hardware Used

Any hardware could be used but known supported hardware so far is:

* [Raspberry pi model 3](https://www.raspberrypi.com/products/raspberry-pi-3-model-b-plus/)
* [Camera](https://www.amazon.com/dp/B07BK1QZ2L?psc=1&ref=ppx_yo2_dt_b_product_details)
* [Microphone](https://www.amazon.com/dp/B074BLM973?psc=1&ref=ppx_yo2_dt_b_product_details)
* [Tripod](https://www.amazon.com/AmazonBasics-Lightweight-Camera-Mount-Tripod/dp/B00XI87KV8/ref=psdc_499310_t1_B00009UT28)

The pi and camera can be mounted to the tripod with zip-ties or velcro straps.

## Software Used

The server page software is included in this repository. For the webrtc capabilities the pi runs the [pion/webrtc](https://github.com/pion/webrtc) system utilizing the hardware decoding capabilities of the pi.

```
sudo apt install hostapd
```

### Hostapd

Hostapd is used to turn the pi into a wireless access point. This enables connecting to the pi directly versus having to go through another network. The configuration used is found at `src/hostapd.conf`.

## Remote WebRTC mode

This is started and stopped from the server page. Will add more setup and usage steps later.

### Building

This project assumes it is cloned to `/home/pi/Projects`.

Change to the `/home/pi/Projects/diy-baby-pi/` directory.

```
go build src/server/webrtc/main.go
```

This will create binary that the node process starts and stops for webrtc through the server.


## Local HDMI Mode

This mode works directly through HDMI for lowest latency and highest reliability. This is a work in progress and support needs to be added to the server page to allow starting and stopping this stream. With the camera and microphone plugged in and a display available on the HDMI port run the following command while logged in through SSH or from the terminal if access to mouse and keyboard is available. I use this with a 100 foot HDMI cable.

```
arecord -f cd - | aplay & raspivid -t 0 -f &
```

Eventually from the server page this will start and stop through press of button. For now it is rudimentary.
