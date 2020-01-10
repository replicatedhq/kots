FROM nginx:1.17.5
ARG version
ARG nginxconf=deploy/nginx.conf

COPY --chown=nginx:root dist /usr/share/nginx/html
COPY --chown=nginx:root ${nginxconf} /etc/nginx/conf.d/default.conf
COPY --chown=nginx:root deploy/root-nginx.conf /etc/nginx/nginx.conf
RUN chown -R nginx:root /usr/share/nginx/html
RUN chown nginx:root /etc/nginx/conf.d
RUN chmod -R g+rw /usr/share/nginx/html

USER nginx

STOPSIGNAL SIGTERM