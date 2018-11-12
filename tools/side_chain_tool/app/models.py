# -*- coding:utf-8 -*-
from . import db



class KeyPair(db.Model):
    __tablename__ = 'key_pair'
    id = db.Column(db.Integer, primary_key=True)
    xprv = db.Column(db.String, nullable=True, unique=True)
    xpub = db.Column(db.String, nullable=True, unique=True)

class PeginAddress(db.Model):
    __tablename__ = 'pegin_address'
    id = db.Column(db.Integer, primary_key=True)
    address = db.Column(db.String, nullable=True, unique=True)
    claim_script = db.Column(db.String, nullable=True, unique=True)

class TxInfo(db.Model):
    __tablename__ = 'tx_info'
    id = db.Column(db.Integer, primary_key=True)
    block_height = db.Column(db.Integer, nullable=True, unique=True)
    txid_main = db.Column(db.String, nullable=True, unique=True)
    txid_claim = db.Column(db.String, nullable=True, unique=True)