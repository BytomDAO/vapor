SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for block_state
-- ----------------------------
DROP TABLE IF EXISTS `block_state`;
CREATE TABLE `block_state`  (
  `height` int(11) NOT NULL,
  `block_hash` varchar(64) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL
) ENGINE = InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for vote
-- ----------------------------
DROP TABLE IF EXISTS `vote`;
CREATE TABLE `vote`  (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `xpub` varchar(128) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `voter_address` varchar(62) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `vote_height` int(11) NOT NULL,
  `vote_num` int(11) NOT NULL,
  `veto_height` int(11) NOT NULL,
  `output_id` varchar(64) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `xpub`(`xpub`, `vote_height`, `output_id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 6 DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
