# kvstore/natsjs

natsjs is a kvstore implementation for go-orb that uses nats with JetStream as the backend.

It's a port of the [store/nats-js](https://github.com/kobergj/plugins/tree/NatsjskvWatcher/v4/store/nats-js) plugin for go-micro to go-orb.

This plugins supports the `kvstore.Watcher` interface!

## Compatibility to "store/nats-js"

The plugin is compatible to the "store/nats-js" plugin for go-micro as long as you don't disable JSONKeyValues or enable BucketPerTable.

The compatiblity is ensured by tests in the [kvstore/natsjs_micro_tests](https://github.com/go-orb/plugins/tree/main/kvstore/natsjs_micro_tests) directory.

## Warning

nats doesn't support per record TTL, so the TTL option per record is ignored.

## Previous Authors

- [kobergj](https://github.com/kobergj)
- [butonic](https://github.com/butonic)
- [Davincible](https://github.com/Davincible)

## Authors

- [jochumdev](https://github.com/jochumdev)

## License

This plugin is Apache 2.0 licensed.