FROM nginx:1.11.8
ARG version
ARG nginxconf=deploy/nginx.conf

COPY dist /usr/share/nginx/html
COPY ${nginxconf} /etc/nginx/conf.d/default.conf

STOPSIGNAL SIGTERM

