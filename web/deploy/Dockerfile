FROM nginx:1.17.5
ARG version
ARG nginxconf=deploy/nginx.conf

COPY --chown=nginx:nginx dist /usr/share/nginx/html
COPY --chown=nginx:nginx ${nginxconf} /etc/nginx/conf.d/default.conf
COPY --chown=nginx:nginx deploy/root-nginx.conf /etc/nginx/nginx.conf

USER nginx

STOPSIGNAL SIGTERM