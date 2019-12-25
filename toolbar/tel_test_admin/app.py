# -*- coding: utf8 -*-
from flask import Flask
from gevent.pywsgi import WSGIServer
from views import tele
app = Flask(__name__)
app.debug = True


app.register_blueprint(tele, url_prefix='')
app.debug = True
http_server = WSGIServer(("0.0.0.0", 5000), app)
http_server.serve_forever()
