/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

CREATE SCHEMA IF NOT EXISTS `federation`;

USE `federation`;

# Dump of table warders
# ------------------------------------------------------------

DROP TABLE IF EXISTS `warders`;

CREATE TABLE `warders` (
  `id` tinyint(1) unsigned NOT NULL AUTO_INCREMENT,
  `pubkey` varchar(64) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pubkey` (`pubkey`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `warders` WRITE;
UNLOCK TABLES;


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
(1,'btm',0,'a75483474799ea1aa6bb910a1a5025b4372bf20bef20f246a2c2dc5e12e8a053','2018-09-13 05:10:43','2018-11-27 09:42:06');

/*!40000 ALTER TABLE `chains` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table cross_transactions
# ------------------------------------------------------------

CREATE TABLE `cross_transactions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `chain_id` int(11) NOT NULL,
  `block_height` int(11) NOT NULL,
  `block_hash` char(64) NOT NULL,
  `tx_index` int(11) NOT NULL,
  `mux_id` char(64) NOT NULL,
  `tx_hash` char(64) NOT NULL,
  `raw_transaction` mediumtext NOT NULL,
  `status` tinyint(1) DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `mux_id` (`mux_id`),
  UNIQUE KEY `tx_hash` (`tx_hash`),
  UNIQUE KEY `raw_transaction` (`raw_transaction`),
  UNIQUE KEY `blockhash_txidx` (`block_hash`,`tx_index`),
  UNIQUE KEY `blockheight_txidx` (`chain_id`,`block_height`,`tx_index`),
  CONSTRAINT `transactions_ibfk_1` FOREIGN KEY (`chain_id`) REFERENCES `chains` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `cross_transactions` WRITE;
UNLOCK TABLES;


# Dump of table cross_transaction_inputs
# ------------------------------------------------------------
CREATE TABLE `cross_transaction_inputs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `tx_id` int(11) NOT NULL,
  `source_pos` int(11) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `input_id` (`tx_id`,`source_pos`),
  CONSTRAINT `cross_transaction_inputs_ibfk_1` FOREIGN KEY (`tx_id`) REFERENCES `cross_transactions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `cross_transaction_inputs` WRITE;
UNLOCK TABLES;


# Dump of table cross_transaction_signs
# ------------------------------------------------------------
CREATE TABLE `cross_transaction_signs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `cross_transaction_id` int(11) NOT NULL,
  `warder_id` int(11) NOT NULL,
  `signatures` text NOT NULL,
  `status` tinyint(1) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `sign_id` (`cross_transaction_id`,`warder_id`),
  CONSTRAINT `cross_transaction_signs_ibfk_1` FOREIGN KEY (`warder_id`) REFERENCES `warders` (`id`),
  CONSTRAINT `cross_transaction_signs_ibfk_1` FOREIGN KEY (`cross_transaction_id`) REFERENCES `cross_transactions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `cross_transaction_signs` WRITE;
UNLOCK TABLES;