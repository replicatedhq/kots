FROM nginx:1.11.8
ARG version
ARG nginxconf=deploy/nginx.conf

COPY dist /usr/share/nginx/html
COPY ${nginxconf} /etc/nginx/conf.d/default.conf

RUN useradd -c 'kotsadm-web user' -m -d /home/kotsadm-web -s /bin/bash -u 1001 kotsadm-web
RUN chown -R kotsadm-web.kotsadm-web /usr/share/nginx/html
RUN chown -R kotsadm-web.kotsadm-web /etc/nginx/conf.d/default.conf
USER kotsadm-web
ENV HOME /home/kotsadm-web

STOPSIGNAL SIGTERM

