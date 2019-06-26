/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

DROP DATABASE `federation`;

CREATE SCHEMA IF NOT EXISTS `federation`;

USE `federation`;


# Dump of table chains
# ------------------------------------------------------------

CREATE TABLE `chains` (
  `id` tinyint(1) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `block_height` int(11) DEFAULT '0',
  `block_hash` char(64) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  UNIQUE KEY `block_hash` (`block_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `chains` WRITE;
/*!40000 ALTER TABLE `chains` DISABLE KEYS */;

INSERT INTO `chains`
(`id`, `name`, `block_height`, `block_hash`, `created_at`, `updated_at`)
VALUES
(1,'bytom',0,'a75483474799ea1aa6bb910a1a5025b4372bf20bef20f246a2c2dc5e12e8a053','2018-09-13 05:10:43','2018-11-27 09:42:06');

/*!40000 ALTER TABLE `chains` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table cross_transactions
# ------------------------------------------------------------

CREATE TABLE `cross_transactions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `chain_id` tinyint(1) NOT NULL,
  `source_block_height` int(11) NOT NULL,
  `source_block_timestamp` int(11) NOT NULL,
  `source_block_hash` char(64) NOT NULL,
  `source_tx_index` int(11) NOT NULL,
  `source_mux_id` char(64) NOT NULL,
  `source_tx_hash` char(64) NOT NULL,
  `source_raw_transaction` mediumtext NOT NULL,
  `dest_block_height` int(11) DEFAULT NULL,
  `dest_block_timestamp` int(11) DEFAULT NULL,
  `dest_block_hash` char(64) DEFAULT NULL,
  `dest_tx_index` int(11) DEFAULT NULL,
  `dest_tx_hash` char(64) DEFAULT NULL,
  `status` tinyint(1) DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `source_mux_id` (`chain_id`,`source_mux_id`),
  UNIQUE KEY `source_tx_hash` (`chain_id`,`source_tx_hash`),
  UNIQUE KEY `source_blockhash_txidx` (`chain_id`,`source_block_hash`,`source_tx_index`),
  UNIQUE KEY `source_blockheight_txidx` (`chain_id`,`source_block_height`,`source_tx_index`),
  UNIQUE KEY `dest_tx_hash` (`chain_id`,`dest_tx_hash`),
  UNIQUE KEY `dest_blockhash_txidx` (`chain_id`,`dest_block_hash`,`dest_tx_index`),
  UNIQUE KEY `dest_blockheight_txidx` (`chain_id`,`dest_block_height`,`dest_tx_index`),
  CONSTRAINT `cross_transactions_ibfk_1` FOREIGN KEY (`chain_id`) REFERENCES `chains` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `cross_transactions` WRITE;
UNLOCK TABLES;


# Dump of table cross_transaction_reqs
# ------------------------------------------------------------
CREATE TABLE `cross_transaction_reqs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `cross_transaction_id` int(11) NOT NULL,
  `source_pos` int(11) NOT NULL,
  `asset_id` int(11) NOT NULL,
  `asset_amount` bigint(20) DEFAULT '0',
  `script` varchar(128) NOT NULL,
  `from_address` varchar(128) NOT NULL DEFAULT '',
  `to_address` varchar(128) NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `req_id` (`cross_transaction_id`,`source_pos`),
  CONSTRAINT `cross_transaction_reqs_ibfk_1` FOREIGN KEY (`cross_transaction_id`) REFERENCES `cross_transactions` (`id`),
  CONSTRAINT `cross_transaction_reqs_ibfk_2` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `cross_transaction_reqs` WRITE;
UNLOCK TABLES;


# Dump of table assets
# ------------------------------------------------------------

CREATE TABLE `assets` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `asset_id` varchar(64) NOT NULL,
  `issuance_program` varchar(64) NOT NULL,
  `vm_version` int(11) NOT NULL DEFAULT '1',
  `raw_definition_byte` text,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `asset_id` (`asset_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `assets` WRITE;
/*!40000 ALTER TABLE `assets` DISABLE KEYS */;

INSERT INTO `assets` (`id`, `asset_id`, `issuance_program`, `vm_version`, `raw_definition_byte`, `created_at`, `updated_at`)
VALUES
  (1,'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff','',1,'7b0a202022646563696d616c73223a20382c0a2020226465736372697074696f6e223a20224279746f6d204f6666696369616c204973737565222c0a2020226e616d65223a202242544d222c0a20202273796d626f6c223a202242544d220a7d','2018-09-13 05:10:43','2018-11-27 09:43:35');

/*!40000 ALTER TABLE `assets` ENABLE KEYS */;
UNLOCK TABLES;