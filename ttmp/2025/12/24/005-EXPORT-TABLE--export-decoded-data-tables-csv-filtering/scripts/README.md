# Scripts (ticket 005)

## Real server validation

- `01-real-export-table.sh`: loads a `.sal`, adds a SPI analyzer from a template, runs `salad export table`, and verifies the CSV output.

### Run

From the `salad/` repo root:

```bash
cd /home/manuel/workspaces/2025-12-27/salad-pass/salad
SAL="/tmp/Session 6.sal" ./ttmp/2025/12/24/005-EXPORT-TABLE--export-decoded-data-tables-csv-filtering/scripts/01-real-export-table.sh
```

### Overrides

You can override host/port/timeout and paths:

```bash
HOST=127.0.0.1 PORT=10430 TIMEOUT=120s \
SAL="/tmp/Session 6.sal" \
SETTINGS_YAML="/home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi.yaml" \
OUT="/tmp/salad-export-table-real.csv" \
./ttmp/2025/12/24/005-EXPORT-TABLE--export-decoded-data-tables-csv-filtering/scripts/01-real-export-table.sh
```


