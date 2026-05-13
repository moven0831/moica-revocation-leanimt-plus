// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

contract LeanIMTPlusRootStorage {
    struct RootInfo {
        uint256 root;
        uint256 crlNumber;
        uint256 updatedAt;
        uint8   depth;
        uint64  leafCount;
    }

    address public relayer;
    mapping(bytes32 => RootInfo) public roots;

    event RootUpdated(
        bytes32 indexed issuerId,
        uint256 root,
        uint256 crlNumber,
        uint8   depth,
        uint64  leafCount
    );

    modifier onlyRelayer() {
        require(msg.sender == relayer, "unauthorized");
        _;
    }

    constructor(address _relayer) {
        relayer = _relayer;
    }

    function setRoot(
        bytes32 issuerId,
        uint256 newRoot,
        uint256 crlNumber,
        uint8   depth,
        uint64  leafCount
    ) external onlyRelayer {
        require(crlNumber > roots[issuerId].crlNumber, "stale CRL");
        roots[issuerId] = RootInfo(newRoot, crlNumber, block.timestamp, depth, leafCount);
        emit RootUpdated(issuerId, newRoot, crlNumber, depth, leafCount);
    }

    function getRoot(bytes32 issuerId) external view returns (uint256) {
        return roots[issuerId].root;
    }

    function getRootInfo(bytes32 issuerId)
        external
        view
        returns (uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
    {
        RootInfo storage info = roots[issuerId];
        return (info.root, info.crlNumber, info.updatedAt, info.depth, info.leafCount);
    }
}
