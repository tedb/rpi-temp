# Raspberry Pi temperature logger for Adafruit IO

RPiTemp is a simple proof-of-concept utility to post to [Adafruit IO](https://adafruit.io)
the readings from one or more [DS18B20](https://www.adafruit.com/product/381) temperature probes.  It's intended for use on a Raspberry Pi.

It has one purpose: on an interval (2 minutes seems right), read the temperatures, and post them using the REST API for Adafruit IO.

## Build and deploy steps

### Format and lint Go code

```shell
goimports -w rpi_temp.go
golangci-lint run  --enable-all rpi_temp.go
```

### Cross-compile for Raspberry Pi

Building a Go binary on the RPi is quite slow, but can be done.  It's recommended
to run the compilation on another computer, then copy just the executable to the RPi, with `scp`.

```
env GOARCH="arm" GOARM="6" GOOS="linux" go build rpi_temp.go
```

TODO(tedb): Convert to a build tool

### Create Adafruit IO "feed"

Substitute each probe's serial number for 011850aaaaaa, from:
`cat /sys/bus/w1/devices/28-*/name`

```shell
export ADAFRUIT_IO_KEY=zzzzzzzzz
export ADAFRUIT_USER=myusername
export SERIAL=011850aaaaaa

curl -X POST -H "Content-Type: application/json" -H "X-AIO-Key: $ADAFRUIT_IO_KEY" \
    --data '{"feed": {"name": "Temp $SERIAL"}}' \
    https://io.adafruit.com/api/v2/$ADAFRUIT_USER/feeds
```

### Run on RPi

In the URL, %s will be substituted with the serial number of the 1-Wire probe.

```
export ADAFRUIT_IO_KEY=zzzzzzzzz
export ADAFRUIT_USER=myusername
export ADAFRUIT_IO_URL=https://io.adafruit.com/api/v2/$ADAFRUIT_USER/feeds/temp-%s/data
./rpi_temp
```

TODO(tedb): document running from systemd timer


## Aren't there several other scripts and tools that do this?

Many scripts and tools are similar.  But, this one is mine. :)

Apparently, writing a tool like this is a [popular hobby](https://github.com/search?q=raspberry+pi+DS18B20).  I didn't find a script that does this exact one thing: read the temperature an post it to Adafruit.IO.  Some of them don't work with multiple temperature probes.

## What's the back story?

After setting up a small fish tank, my first realization was it would
be a perfect excuse to rig up an old otherwise-useless first-generation Raspberry Pi.
Rigging up a fish-tank cam with some 1-Wire probes is pretty much the Hello World
of RPi electronics tinkering.  This is my idea of "a fun hobby".

## Why Adafruit IO?

Adafruit has a fantastic focus on serving the
individual hobby hacker for electronics components, microcontrollers, etc.
I am primarily a software person,
and over the years I've been grateful to have learned so much from Adafruit's hardware tutorials.
There are places to get cheaper prices on parts, but it's worthwhile
to know that Adafruit vets the things they sell, and that I can trust the
parts are legitimate and will work as intended.

In a cunning move, they have launched an "internet of things" cloud service
targeting my exact use case.  I could also get started with the API immediately
with zero fuss.  The first prototype was just a Bash one-liner with `curl`.

## What about the major cloud providers?

The major cloud providers (Google Cloud, AWS, Azure, etc.) all have big pushes into
the "Internet of Things" space.  Unfortunately, it was harder to figure out
how to get started for my simple use case than I was willing to invest.
Just setting up authentication for a device (with OAuth, certificates, etc.)
is a bigger lift than the actual work to log the temperatures.

## Goals?

This is part utilitarian tool, and part "art project": in that way, it is not unlike gardening.  Potential future directions:

- [x] Automatically build a binary for RPi using [GitHub Actions](https://github.com/features/actions)
- [ ] ~~Refactor the REST call to use the official [Go client](https://github.com/adafruit/io-client-go#usage).~~
- [x] Use MQTT client to post results
- [ ] Integration tests.
- [x] On start-up, create data feeds as needed for the attached sensors. (MQTT group post automatically creates feeds)
- [x] Provide systemd unit files
- [ ] Run the build (and dev tools like linting) with [Bazel](https://bazel.build/)
- [ ] Provide Debian package, from Bazel
- [ ] Notify me (e.g. email, or [Pushover](https://pushover.net/)) when the temp exceeds a range
- [ ] Maybe port the code to [Elixir](https://elixir-lang.org/)
- [x] Consider ["bulk read"](https://www.kernel.org/doc/html/latest/w1/slaves/w1_therm.html) interface (faster?)
- [ ] Capture RPi camera and publish to Adafruit for its dashboard; see [docs](https://io.adafruit.com/api/docs/cookbook.html#publishing-image-data)
