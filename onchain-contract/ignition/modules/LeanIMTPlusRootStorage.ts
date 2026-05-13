import { buildModule } from "@nomicfoundation/hardhat-ignition/modules";

export default buildModule("LeanIMTPlusRootStorageModule", (m) => {
  const relayer = m.getParameter("relayer");
  const leanimtPlusRootStorage = m.contract("LeanIMTPlusRootStorage", [relayer]);
  return { leanimtPlusRootStorage };
});
