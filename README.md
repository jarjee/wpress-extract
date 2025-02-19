<div align="center">
  <h1>wpress-extract</h1>
  <p>
    A simple CLI tool to unpack <code>.wpress</code> files generated by the <a href="https://wordpress.org/plugins/all-in-one-wp-migration/" target="_blank" rel="noopener">All-in-One WP Migration</a> Wordpress plugin.
  </p>
  <hr />
  <p>
    <img src="https://imgs.xkcd.com/comics/standards.png" alt="A funny comic from xkcd.com about creating new standards in computer industry" />
    <br />
    <sub>"Standards" by <a href="https://xkcd.com/927/">xkcd</a>.</sub>
  </p>
</div>

## Usage

### Build from Source
```sh
git clone https://github.com/jarjee/wpress-extract
cd wpress-extract
make build
sudo install bin/wpress-extract /usr/local/bin
```

### Extract Archives
```sh
# Extract a wpress file
./bin/wpress-extract -input your-migration.wpress

# Specify custom output directory
./bin/wpress-extract -input your-migration.wpress -out ./output-dir

# Force overwrite existing directory
./bin/wpress-extract -input your-migration.wpress -force
```

The command creates a new directory with the same name (e.g. `your-migration/`) where it extracts the archive contents.

### Options

| Option            | Description                                                                           |
| ----------------- | ------------------------------------------------------------------------------------- |
| `-o, --out <dir>` | Define an alternate directory where the archive should be extracted to.               |
| `-f, --force`     | Skip the check if the output directory already exists and override the content in it. |

## Acknowledgements

The functionality of this package is inspired by the [Wpress-Extractor](https://github.com/fifthsegment/Wpress-Extractor) tool.
This fork contains modifications assisted by [aider](https://aider.chat), an LLM-powered coding assistant.

## Maintainers

<!-- prettier-ignore-start -->

| [<img src="https://avatars0.githubusercontent.com/u/472867?v=4" width="100px;"/><br /><sub><b>Felix Haus</b></sub>](https://github.com/ofhouse)<br /><sub>[Website](https://felix.house/) • [Twitter](https://twitter.com/ofhouse)</sub> | [<img src="https://avatars.githubusercontent.com/u/jarjee" width="100px;"/><br /><sub><b>@jarjee</b></sub>](https://github.com/jarjee)<br /><sub>Maintainer</sub> |
| :---: | :---: |

<!-- prettier-ignore-end -->

## License

MIT - see [LICENSE](./LICENSE) for details.
