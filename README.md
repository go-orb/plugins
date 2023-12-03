# github.com/go-orb/go-orb -> plugins

This repo contains plugins for [github.com/go-orb/go-orb](https://github.com/go-orb/go-orb).

## WIP

This project is a work in progress, please do not use yet!

## Community

- Chat with us on [Discord](https://discord.gg/sggGS389qb)

## Development

### go.mod replacements

When you work on packages that require changes in the core you will need:

```bash
go-task mod-replace
```

This will add replace statements to all go.mod's in this repo.

**Before** a git commit you have to:

```bash
go-task mod-dropreplace
```

We do not accept commit's with a "replace" line.

### Quirks

#### It's not allowed to import plugins in github.com/go-orb/go-orb

To prevent import cycles it's not allowed to import plugins in github.com/go-orb/go-orb.

## Authors

- [David Brouwer](https://github.com/Davincible/)
- [René Jochum](https://github.com/jochumdev)

## License

go-orb is Apache 2.0 licensed same as go-micro.
