# Redline Parser

This script processes the output folder of Redline, extracting various types of data and converting them into Parquet files. The data includes user information, passwords, autofills, installed software, processes, and cookies. Each type of data is stored in a separate Parquet file with a specific schema.

### Unique ID Generation

The unique ID for each entry is generated using the following format: `HWID_DATE`. The `HWID` is either read from the user information or generated if not present. The `DATE` is derived from the log date in the user information or the current date if the log date is not available.

### Flags and Parameters

- `-input_dir`: The input folder path containing the Redline output files.
- `-output_dir`: The output folder path where the Parquet files will be saved. Default is the current directory.

### Parquet File Schemas

#### UserInfo
- `unique_id`: BYTE_ARRAY, UTF8
- `build_id`: BYTE_ARRAY, UTF8
- `ip`: BYTE_ARRAY, UTF8
- `file_location`: BYTE_ARRAY, UTF8
- `username`: BYTE_ARRAY, UTF8
- `country`: BYTE_ARRAY, UTF8
- `zip_code`: BYTE_ARRAY, UTF8
- `location`: BYTE_ARRAY, UTF8
- `hwid`: BYTE_ARRAY, UTF8
- `current_lang`: BYTE_ARRAY, UTF8
- `screen_size`: BYTE_ARRAY, UTF8
- `time_zone`: BYTE_ARRAY, UTF8
- `os`: BYTE_ARRAY, UTF8
- `uac`: BYTE_ARRAY, UTF8
- `process_elevation`: BYTE_ARRAY, UTF8
- `elevated`: BOOLEAN
- `log_date`: INT64, TIMESTAMP_MILLIS
- `available_keyboard_layouts`: MAP, LIST, BYTE_ARRAY, UTF8
- `hardwares`: MAP, LIST, BYTE_ARRAY, UTF8
- `antiviruses`: MAP, LIST, BYTE_ARRAY, UTF8
- `additional_fields`: MAP, MAP, BYTE_ARRAY, UTF8

#### PasswordEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `url`: BYTE_ARRAY, UTF8
- `username`: BYTE_ARRAY, UTF8
- `password`: BYTE_ARRAY, UTF8
- `application`: BYTE_ARRAY, UTF8

#### AutofillEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `key`: BYTE_ARRAY, UTF8
- `value`: BYTE_ARRAY, UTF8

#### CreditCardEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `holder`: BYTE_ARRAY, UTF8
- `type`: BYTE_ARRAY, UTF8
- `number`: BYTE_ARRAY, UTF8
- `expiry`: BYTE_ARRAY, UTF8

#### InstalledSoftwareEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `id`: INT64
- `name`: BYTE_ARRAY, UTF8
- `version`: BYTE_ARRAY, UTF8

#### ProcessEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `id`: INT64
- `name`: BYTE_ARRAY, UTF8
- `cmdline`: BYTE_ARRAY, UTF8

#### CookieEntry
- `unique_id`: BYTE_ARRAY, UTF8
- `domain`: BYTE_ARRAY, UTF8
- `flag`: BOOLEAN
- `path`: BYTE_ARRAY, UTF8
- `secure`: BOOLEAN
- `expiration`: INT64
- `name`: BYTE_ARRAY, UTF8
- `value`: BYTE_ARRAY, UTF8

### Bash Script Helpers

To process multiple Redline output folders at scale, you can use the following bash script:

```bash
#!/bin/bash

input_base_dir="/path/to/redline/outputs"
output_base_dir="/path/to/parquet/outputs"

for input_dir in "$input_base_dir"/*; do
  if [ -d "$input_dir" ]; then
    output_dir="$output_base_dir/$(basename "$input_dir")"
    mkdir -p "$output_dir"
    ./redline_parser -input_dir "$input_dir" -output_dir "$output_dir"
  fi
done
```

### DuckDB Queries

You can use DuckDB to query the generated Parquet files. Here are some example queries:

```sql
-- Load the UserInfo Parquet file
SELECT * FROM 'user_info_*.parquet';

-- Count the number of passwords
SELECT COUNT(*) FROM 'passwords_*.parquet';

-- List all installed software
SELECT name, version FROM 'installed_software_*.parquet';

-- Find all processes with a specific name
SELECT * FROM 'processes_*.parquet' WHERE name = 'chrome.exe';

-- Get all cookies for a specific domain
SELECT * FROM 'cookies_*.parquet' WHERE domain = '.example.com';
```

### Running the Script

To run the script, use the following command:

```bash
./redline_parser -input_dir /path/to/redline/output -output_dir /path/to/parquet/output
```

This will process the Redline output folder and generate the corresponding Parquet files in the specified output directory.