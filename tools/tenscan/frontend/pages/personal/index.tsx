import React from "react";
import Layout from "@/src/components/layouts/default-layout";
import { Metadata } from "next";
import PersonalTransactions from "@/src/components/modules/personal";
import { useWalletConnection } from "@/src/components/providers/wallet-provider";
import ConnectWalletButton from "@repo/ui/common/connect-wallet";
import EmptyState from "@repo/ui/common/empty-state";
import { ethereum } from "@repo/ui/lib/utils";

export const metadata: Metadata = {
  title: "Personal Transactions",
  description: "Tenscan Personal Transactions",
};

export default function PersonalPage() {
  const { walletConnected, walletAddress, connectWallet, disconnectWallet } =
    useWalletConnection();

  return (
    <Layout>
      {walletConnected ? (
        <PersonalTransactions />
      ) : (
        <EmptyState
          title="Connect Wallet"
          description="Connect your wallet to view your personal transactions."
          action={
            <ConnectWalletButton
              text={
                ethereum
                  ? "Connect Wallet to continue"
                  : "Install MetaMask to continue"
              }
              walletConnected={walletConnected}
              walletAddress={walletAddress}
              connectWallet={connectWallet}
              disconnectWallet={disconnectWallet}
            />
          }
        />
      )}
    </Layout>
  );
}
