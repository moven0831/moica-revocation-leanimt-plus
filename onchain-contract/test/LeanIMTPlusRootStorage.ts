import { expect } from "chai";
import { network } from "hardhat";

const { ethers } = await network.connect();

describe("LeanIMTPlusRootStorage", function () {
  const ISSUER_G2 = ethers.keccak256(ethers.toUtf8Bytes("MOICA-G2"));
  const ISSUER_G3 = ethers.keccak256(ethers.toUtf8Bytes("MOICA-G3"));

  async function deployFixture() {
    const [owner, relayer, other] = await ethers.getSigners();
    const contract = await ethers.deployContract("LeanIMTPlusRootStorage", [
      relayer.address,
    ]);
    return { contract, owner, relayer, other };
  }

  describe("Deployment", function () {
    it("should set the relayer address", async function () {
      const { contract, relayer } = await deployFixture();
      expect(await contract.relayer()).to.equal(relayer.address);
    });
  });

  describe("setRoot", function () {
    it("should update root, crl, timestamp, depth, leafCount", async function () {
      const { contract, relayer } = await deployFixture();
      const root = 12345n;
      const crlNumber = 100n;
      const depth = 19;
      const leafCount = 412345n;

      await contract
        .connect(relayer)
        .setRoot(ISSUER_G2, root, crlNumber, depth, leafCount);

      const [
        storedRoot,
        storedCrlNumber,
        storedUpdatedAt,
        storedDepth,
        storedLeafCount,
      ] = await contract.getRootInfo(ISSUER_G2);
      expect(storedRoot).to.equal(root);
      expect(storedCrlNumber).to.equal(crlNumber);
      expect(storedUpdatedAt).to.be.greaterThan(0n);
      expect(storedDepth).to.equal(BigInt(depth));
      expect(storedLeafCount).to.equal(leafCount);
    });

    it("should revert on stale CRL number", async function () {
      const { contract, relayer } = await deployFixture();

      await contract.connect(relayer).setRoot(ISSUER_G2, 111n, 100n, 5, 32n);

      await expect(
        contract.connect(relayer).setRoot(ISSUER_G2, 222n, 50n, 5, 32n),
      ).to.be.revertedWith("stale CRL");

      await expect(
        contract.connect(relayer).setRoot(ISSUER_G2, 222n, 100n, 5, 32n),
      ).to.be.revertedWith("stale CRL");
    });

    it("should revert for non-relayer", async function () {
      const { contract, other } = await deployFixture();

      await expect(
        contract.connect(other).setRoot(ISSUER_G2, 111n, 100n, 5, 32n),
      ).to.be.revertedWith("unauthorized");
    });

    it("should emit RootUpdated event with all fields", async function () {
      const { contract, relayer } = await deployFixture();
      const root = 99999n;
      const crlNumber = 42n;
      const depth = 7;
      const leafCount = 128n;

      await expect(
        contract
          .connect(relayer)
          .setRoot(ISSUER_G2, root, crlNumber, depth, leafCount),
      )
        .to.emit(contract, "RootUpdated")
        .withArgs(ISSUER_G2, root, crlNumber, depth, leafCount);
    });

    it("should support multiple issuer IDs independently", async function () {
      const { contract, relayer } = await deployFixture();

      await contract.connect(relayer).setRoot(ISSUER_G2, 111n, 10n, 4, 16n);
      await contract.connect(relayer).setRoot(ISSUER_G3, 222n, 20n, 5, 32n);

      expect(await contract.getRoot(ISSUER_G2)).to.equal(111n);
      expect(await contract.getRoot(ISSUER_G3)).to.equal(222n);

      await contract.connect(relayer).setRoot(ISSUER_G2, 333n, 11n, 4, 17n);
      expect(await contract.getRoot(ISSUER_G2)).to.equal(333n);
      expect(await contract.getRoot(ISSUER_G3)).to.equal(222n);
    });
  });

  describe("getRoot", function () {
    it("should return zero for uninitialized issuer", async function () {
      const { contract } = await deployFixture();
      const unknownId = ethers.keccak256(ethers.toUtf8Bytes("UNKNOWN"));
      expect(await contract.getRoot(unknownId)).to.equal(0n);
    });

    it("should return the latest root after updates", async function () {
      const { contract, relayer } = await deployFixture();

      await contract.connect(relayer).setRoot(ISSUER_G2, 100n, 1n, 1, 2n);
      expect(await contract.getRoot(ISSUER_G2)).to.equal(100n);

      await contract.connect(relayer).setRoot(ISSUER_G2, 200n, 2n, 2, 4n);
      expect(await contract.getRoot(ISSUER_G2)).to.equal(200n);
    });
  });

  describe("getRootInfo", function () {
    it("should return zeros for uninitialized issuer", async function () {
      const { contract } = await deployFixture();
      const unknownId = ethers.keccak256(ethers.toUtf8Bytes("UNKNOWN"));
      const [root, crlNumber, updatedAt, depth, leafCount] =
        await contract.getRootInfo(unknownId);
      expect(root).to.equal(0n);
      expect(crlNumber).to.equal(0n);
      expect(updatedAt).to.equal(0n);
      expect(depth).to.equal(0n);
      expect(leafCount).to.equal(0n);
    });
  });
});
