# Analyzer templates

These templates are **our conventions** (not Saleae API schemas). They exist to make `salad analyzer add` reproducible.

## Usage

Start a capture, then add an analyzer using a template:

```bash
salad analyzer add --capture-id <id> --name "SPI" --label "spi" --settings-yaml /abs/path/to/configs/analyzers/spi.yaml
```

## Included templates (initial pack)

- `spi.yaml`
- `i2c.yaml`
- `async-serial.yaml`

Override any template key using typed overrides (recommended):

```bash
salad analyzer add --capture-id <id> --name "SPI" --label "spi" \
  --settings-yaml /abs/path/to/configs/analyzers/spi.yaml \
  --set-int "Clock=0" --set-int "MOSI=1" --set-int "MISO=2" --set-int "Enable=3"
```


