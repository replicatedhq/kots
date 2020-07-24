#!/bin/bash
set -e

for i in "$@"
do
case $i in
    -u=*|--uuid=*)
    UUID="${i#*=}"
    ;;
    *)
esac
done

export UUID=${UUID:-`id -u -n`}

# Generate fixtures
mkdir -p -m 755 $PWD/migrations/fixtures/schema
docker run \
  -v $PWD/migrations/fixtures:/out \
  -v $PWD/migrations/tables:/in \
  schemahero/schemahero:0.9.0 \
  fixtures --input-dir /in --output-dir /out/schema --dbname ship-cloud --driver postgres

printf "\n\n\n"
printf "These images are good for 2 hours\n"
