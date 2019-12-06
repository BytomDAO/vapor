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
  UNIQUE KEY `block_hash` (`id`,`block_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `chains` WRITE;
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
  KEY `chain_id` (`chain_id`),
  UNIQUE KEY `source_tx_hash` (`source_tx_hash`),
  UNIQUE KEY `dest_tx_hash` (`dest_tx_hash`),
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
  `issuance_program` mediumtext NOT NULL,
  `vm_version` int(11) NOT NULL DEFAULT '1',
  `definition` text,
  `is_open_federation_issue` tinyint(1) DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `asset_id` (`asset_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `assets` WRITE;
UNLOCK TABLES;