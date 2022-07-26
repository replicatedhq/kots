#!/bin/bash
sed 's/localhost:8800/###_HOSTNAME_###/g' /web/dist/index.html > /tmp/index_html_edit && cat /tmp/index_html_edit > /web/dist/index.html && rm /tmp/index_html_edit
sed 's/http:/https:/g' /web/dist/index.html > /tmp/index_html_edit && cat /tmp/index_html_edit > /web/dist/index.html && rm /tmp/index_html_edit

/kotsadm api