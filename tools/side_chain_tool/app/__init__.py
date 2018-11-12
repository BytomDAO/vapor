# -*- coding:utf-8 -*-
from os import path, environ
from flask import Flask, request
from flask_bootstrap import Bootstrap
from flask_sqlalchemy import SQLAlchemy
from werkzeug.routing import BaseConverter
from config import conifg
import subprocess


class RegexConverter(BaseConverter):
    def __init__(self, url_map, *items):
        super(RegexConverter, self).__init__(url_map)
        self.regex = items[0]

bootstrap = Bootstrap()
db = SQLAlchemy()
basedir = path.abspath(path.dirname(__file__))

# 启动主链
def startmainchain(datadir, args=""):
    shutil.rmtree(datadir)
    subprocess.Popen(("bytom init -r="+datadir+" "+args).split(), stdout=subprocess.PIPE)
    subprocess.Popen(("bytom node -r="+datadir+" "+args).split(), stdout=subprocess.PIPE)

# 启动侧链
#./bytomd.exe node -r "side_chain" --validate_pegin true --side.fedpeg_xpubs "227e08f80ebc11bc0406ffe9f941b117a0259dfbae3c266f96030ddb7d89760d33b9007a5d5d25c95b34650ad3a4c830fc5dcf96828820b7d98e6d7070f835c2"  --side.sign_block_xpubs "227e08f80ebc11bc0406ffe9f941b117a0259dfbae3c266f96030ddb7d89760d33b9007a5d5d25c95b34650ad3a4c830fc5dcf96828820b7d98e6d7070f835c2" --signer "581ffdbd66a895ba28561d0931e93857e253372b465549aa22f94830118e2a4633b9007a5d5d25c95b34650ad3a4c830fc5dcf96828820b7d98e6d7070f835c2" --side.parent_genesis_block_hash "a97a7a59e0e313f9300a2d7296336303889930bfdf5a80d8a9b05db343c03380"
def startsidechain(datadir, args=""):
    shutil.rmtree(datadir)
    subprocess.Popen((ELEMENTSPATH+"/bytom_side init -r="+datadir+" "+args).split(), stdout=subprocess.PIPE)
    subprocess.Popen((ELEMENTSPATH+"/bytom_side node -r="+datadir+" "+args).split(), stdout=subprocess.PIPE)

def create_app(config_name):
    app = Flask(__name__)
    app.config.from_object(conifg[config_name])   # 配置都在config.py这个文件中
    conifg[config_name].init_app(app)

    bootstrap.init_app(app)
    db.init_app(app)

    from .api import api as api_blueprint
    app.register_blueprint(api_blueprint, url_prefix='/api')
    #app.register_blueprint(api_blueprint, static_folder='static', template_folder='templates')

    return app

