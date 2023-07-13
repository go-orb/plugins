# github.com/go-orb/go-orb -> plugins

This repo contains plugins for [github.com/go-orb/go-orb](https://github.com/go-orb/go-orb).

## WIP

This project is a work in progress, please do not use yet!

## Development

### go.mod replacements

When you work on packages that require changes in other plugins or changes on the core you will need:

```bash
go-task mod-replace
```

This will add replace statements to all go.mod's in this repo.

**Before** a git commit you have to:

```bash
go-task mod-dropreplace
```

We do not accept commit's with a "replace" line outside of "github.com/go-orb/plugins/".

### Quirks

#### github.com/go-orb/go-orb is not allowed to import plugins from here

To prevent import cycles it's not allowed to import in github.com/go-orb/go-orb plugins from here.

## Authors

- [David Brouwer](https://github.com/Davincible/)
- [René Jochum](https://github.com/jochumdev)

## License

go-orb is Apache 2.0 licensed same as go-micro.
