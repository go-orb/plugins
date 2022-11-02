# HTTP Server Plugin

An HTTP server provider with HTTP3 support build in (not enabled by default).

Default router used is [Chi](https://github.com/go-chi/chi). You can use another
router if you want, but you will need to write a plugin for it to support the 
router interface used.

