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
  schemahero/schemahero:0.9 \
  fixtures --input-dir /in --output-dir /out/schema --dbname ship-cloud --driver postgres

make -C migrations/fixtures deps build run build-ttl-ci.sh
make -C migrations build-ttl-ci.sh
make -C web deps build-kotsadm
make kotsadm build-ttl-ci.sh
make -C operator build build-ttl-ci.sh
make -C kurl_proxy build build-ttl-ci.sh
make -C api no-yarn deps build build-ttl-ci.sh
make -C minio build-ttl-ci.sh

printf "\n\n\n"
printf "These images are good for 2 hours\n"
