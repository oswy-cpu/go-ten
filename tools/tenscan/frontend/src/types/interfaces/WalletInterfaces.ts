import { ethers } from "ethers";

export interface WalletConnectionContextType {
  provider: ethers.providers.Web3Provider | null;
  walletConnected: boolean;
  isWrongNetwork: boolean;
  walletAddress: string | null;
  connectWallet: () => Promise<void>;
  disconnectWallet: () => void;
  switchNetwork: () => void;
}

export interface Props {
  children: React.ReactNode;
}

export interface State {
  hasError: boolean;
}

export interface WalletConnectionProviderProps {
  children: React.ReactNode;
}
