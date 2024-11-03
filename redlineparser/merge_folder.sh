#!/bin/sh

# this script uses clickhouse-local to merge different parquet file sets into a few

prefixes="passwords user_info cookies autofills processes installed_software"

# for each prefix, there are 1000s of parquet files in the format of prefix_*.parquet. they can all be merged into one with the output of all_PREFIX.parquet
#  SELECT * FROM file('PREFIX*.parquet', Parquet) INTO OUTFILE 'all_PREFIX.parquet' FORMAT Parquet;

for prefix in $prefixes
do
    clickhouse-local --query="SELECT * FROM file('$prefix*.parquet', Parquet) INTO OUTFILE 'all_$prefix.parquet' FORMAT Parquet;"
done

# to clean up the folder, you can move all the all_* files into a new folder, remove all the parquet files, and move the all_* files back to the root folder
mkdir aggregated && mv all_* aggregated/ && rm *.parquet && mv aggregated/* . && rm -rf aggregated

# now that each folder has a few parquet files, you can merge the similar ones in different directories into one
# for prefix in $prefixes
# do
#     clickhouse-local --query="SELECT * FROM file('$prefix*.parquet', Parquet) INTO OUTFILE 'all_$prefix.parquet' FORMAT Parquet;"
# done