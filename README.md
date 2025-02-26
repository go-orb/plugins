# github.com/go-orb/go-orb -> plugins

This repo contains plugins for [github.com/go-orb/go-orb](https://github.com/go-orb/go-orb).

## Community

- Chat with us on [Discord](https://discord.gg/sggGS389qb)

## Development

We do not accept commit's with a "replace" line in a go.mod.

### Run the tests

Install [dagger](https://docs.dagger.io/quickstart/cli)

```sh
dagger call test --root=.
```

### Check linting

```sh
dagger call lint --root=.
```

#### Run the tests for a single plugin

```sh
cd server/http
go test ./... -v -race -cover
cd ...
```

or with dagger

```sh
dagger call test --root=./server/http
```

### Quirks

#### It's not allowed to import plugins in github.com/go-orb/go-orb

To prevent import cycles it's not allowed to import plugins in github.com/go-orb/go-orb.

## Authors

- [David Brouwer](https://github.com/Davincible/)
- [René Jochum](https://github.com/jochumdev)

## License

go-orb is Apache 2.0 licensed same as go-micro.
