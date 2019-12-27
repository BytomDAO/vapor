# -*- coding: utf8 -*-
from flask import Flask
from gevent.pywsgi import WSGIServer
from views import tele
from flask_cors import CORS
app = Flask(__name__)
CORS(app, supports_credentials=True)

app.debug = True


app.register_blueprint(tele, url_prefix='')
app.debug = True
http_server = WSGIServer(("0.0.0.0", 5000), app)
http_server.serve_forever()
