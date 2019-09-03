/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

CREATE SCHEMA IF NOT EXISTS `precognitive`;
DROP DATABASE `precognitive`;
CREATE SCHEMA `precognitive`;

USE `precognitive`;

# Dump of table nodes
# ------------------------------------------------------------

CREATE TABLE `nodes` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `alias` varchar(128) NOT NULL DEFAULT '',
  `xpub` char(128) NOT NULL DEFAULT '',
  `public_key` char(64) NOT NULL DEFAULT '',
  `ip` varchar(128) NOT NULL DEFAULT '',
  `port` smallint unsigned NOT NULL DEFAULT '0',
  `best_height` int(11) DEFAULT '0',
  `avg_rtt_ms` int(11) DEFAULT NULL,
  `latest_daily_uptime_minutes` int(11) DEFAULT '0',
  `status` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `address` (`ip`,`port`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `nodes` WRITE;
UNLOCK TABLES;


# Dump of table node_livenesses
# ------------------------------------------------------------

CREATE TABLE `node_livenesses` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `node_id` int(11) NOT NULL,
  `ping_times` int(11) DEFAULT '0',
  `pong_times` int(11) DEFAULT '0',
  `best_height` int(11) DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  CONSTRAINT `node_livenesses_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `node_livenesses` WRITE;
UNLOCK TABLES;

