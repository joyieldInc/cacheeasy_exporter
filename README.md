# cacheeasy_exporter
Simple server that scrapes Machine/Redis/Predixy stats and
exports them via HTTP for Prometheus consumption

## Build

It is as simple as:

    $ make

## Configuration
see config file cacheeasy_exporter.yml

    bind: <addr> # default is :9123

    predixy:
        - <addr> <name>
        ...
          
    redis:
        - <addr> <name>
        ...

cacheeasy_exporter reload config file every 10 seconds,
the predixy and redis servers will be refresh

## Running

    $ ./cacheeasy_exporter

To change default options, see:

    $ ./cacheeasy_exporter --help

## License

Copyright (C) 2017 Joyield, Inc. <joyield.com#gmail.com>

All rights reserved.

License under BSD 3-clause "New" or "Revised" License
