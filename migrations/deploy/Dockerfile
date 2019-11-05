FROM schemahero/schemahero:alpha

ADD ./tables ./tables

RUN useradd -c 'kotsadm-migrations user' -m -d /home/kotsadm-migrations -s /bin/bash -u 1001 kotsadm-migrations
RUN chown -R kotsadm-migrations.kotsadm-migrations ./tables
USER kotsadm-migrations
ENV HOME /home/kotsadm-migrations

CMD ["apply"]
