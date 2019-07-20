/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

USE `federation`;

# init table chains
# ------------------------------------------------------------

LOCK TABLES `chains` WRITE;
/*!40000 ALTER TABLE `chains` DISABLE KEYS */;

INSERT INTO `chains`
(`id`, `name`, `block_height`, `block_hash`, `created_at`, `updated_at`)
VALUES
(1,'btm',0,'ce4fe9431cd0225b3a811f8f8ec922f2b07a921bb12a8dddae9a85540072c770','2018-09-13 05:10:43','2018-11-27 09:42:06');

INSERT INTO `chains`
(`id`, `name`, `block_height`, `block_hash`, `created_at`, `updated_at`)
VALUES
(2,'vapor',0,'242d1a7fa296b68b0eb15a0958ae57b979fa1dc2a6cc945778628af91ec15a9e','2018-09-13 05:10:43','2018-11-27 09:42:06');

/*!40000 ALTER TABLE `chains` ENABLE KEYS */;
UNLOCK TABLES;


# init table assets
# ------------------------------------------------------------

LOCK TABLES `assets` WRITE;
/*!40000 ALTER TABLE `assets` DISABLE KEYS */;

INSERT INTO `assets` (`id`, `asset_id`, `issuance_program`, `vm_version`, `definition`, `created_at`, `updated_at`)
VALUES
  (1,'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff','',1,
  '{
    "decimals": 8,
    "description": "Bytom Official Issue",
    "name": "BTM",
    "symbol": "BTM"
  }',
  '2018-09-13 05:10:43','2018-11-27 09:43:35');

/*!40000 ALTER TABLE `assets` ENABLE KEYS */;
UNLOCK TABLES;