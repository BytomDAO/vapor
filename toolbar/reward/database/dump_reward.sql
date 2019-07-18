SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for block_state
-- ----------------------------
DROP TABLE IF EXISTS `block_states`;
CREATE TABLE `block_states`  (
  `height` int(11) NOT NULL,
  `block_hash` varchar(64) NOT NULL
) ENGINE = InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for vote
-- ----------------------------
DROP TABLE IF EXISTS `utxos`;
CREATE TABLE `utxos`  (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `xpub` varchar(128) NOT NULL,
  `voter_address` varchar(62) NOT NULL,
  `vote_height` int(11) NOT NULL,
  `vote_num` bigint(21) NOT NULL,
  `veto_height` int(11) NOT NULL,
  `output_id` varchar(64) NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `xpub`(`xpub`, `vote_height`, `output_id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 6 DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
