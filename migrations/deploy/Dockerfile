FROM migrate/migrate
ADD migrations /migrations
ARG version=unknown
LABEL version=${version}
ENTRYPOINT []
CMD [ \
  "/bin/sh", \
  "-c", \
  "/migrate -path=/migrations -database=${POSTGRES_URI} up" \
]
