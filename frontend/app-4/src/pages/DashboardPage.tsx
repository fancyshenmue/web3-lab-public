import { useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ethers } from 'ethers'

type MenuSection = 'profile' | 'deploy' | 'mint' | 'transfer';

type LinkedIdentity = {
  identity_id: string;
  provider_id: string;
  provider_user_id: string;
  display_name?: string;
  is_primary: boolean;
  linked_at: string;
  scw_address?: string;
};
type TokenType = 'ERC20' | 'ERC721' | 'ERC1155';

type TokenBalance = {
  address: string;
  decimals: number;
  symbol: string;
  name: string;
  balanceRaw: string;
  type: string;
  id?: string;
  isOwner?: boolean;
};

type ContextMenuType = 'address' | 'tx' | 'token' | 'generic';
type ContextMenuData = { x: number; y: number; value: string; type: ContextMenuType } | null;

const CopyableFieldDark = ({ label, value, isMono = false, onContextMenu }: { label: string, value: string | boolean, isMono?: boolean, onContextMenu?: (e: React.MouseEvent, val: string) => void }) => {
  const [copied, setCopied] = useState(false)
  const stringValue = typeof value === 'boolean' ? (value ? 'true' : 'false') : String(value)

  const handleCopy = () => {
    navigator.clipboard.writeText(stringValue)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleRightClick = (e: React.MouseEvent) => {
    if (onContextMenu) {
      onContextMenu(e, stringValue)
    }
  }

  return (
    <div
      className="px-4 py-2.5 flex items-start gap-3 hover:bg-white/5 transition-colors group cursor-pointer rounded-lg"
      onClick={handleCopy}
      onContextMenu={handleRightClick}
      title="Click to copy · Right-click for more"
    >
      <span className="text-[10px] font-bold text-gray-500 uppercase tracking-widest min-w-[90px] pt-0.5">{label}</span>
      <span className={`text-sm text-gray-300 break-all flex-1 ${isMono ? 'font-mono text-violet-300' : ''}`}>
        {stringValue}
      </span>
      <span className={`text-xs font-medium transition-opacity flex-shrink-0 ${copied ? 'text-green-400 opacity-100' : 'text-violet-400 opacity-0 group-hover:opacity-100'}`}>
        {copied ? '✓' : 'Copy'}
      </span>
    </div>
  )
}

export const DashboardPage = () => {
  const navigate = useNavigate()
  const [token, setToken] = useState<string | null>(null)
  const { gatewayUrl } = (window as any).__RUNTIME_CONFIG__ || { gatewayUrl: 'https://gateway.web3-local-dev.com' }

  const [userInfo, setUserInfo] = useState<any>(null)
  const [smartWalletAddress, setSmartWalletAddress] = useState<string | null>(null)
  const [walletAddress, setWalletAddress] = useState<string | null>(null)
  const [txLoading, setTxLoading] = useState(false)
  const [txResult, setTxResult] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [txCopied, setTxCopied] = useState(false)
  const [scwCopied, setScwCopied] = useState(false)

  // Portfolio State
  const [portfolio, setPortfolio] = useState<TokenBalance[]>([])
  const [portfolioLoading, setPortfolioLoading] = useState(false)
  const [selectedAsset, setSelectedAsset] = useState<TokenBalance | null>(null)
  const [amountError, setAmountError] = useState<string | null>(null)

  // Layout State
  const [activeSection, setActiveSection] = useState<MenuSection>('deploy')
  const [activeToken, setActiveToken] = useState<TokenType>('ERC20')

  // Profile State
  const [switchStatus, setSwitchStatus] = useState<string | null>(null)
  const [tokenCopied, setTokenCopied] = useState(false)
  const [addressCopied, setAddressCopied] = useState(false)
  const [showToken, setShowToken] = useState(false)

  // Linked Identities State
  const [linkedIdentities, setLinkedIdentities] = useState<LinkedIdentity[]>([])
  const [identitiesLoading, setIdentitiesLoading] = useState(false)
  const [linkingWallet, setLinkingWallet] = useState(false)
  const [linkError, setLinkError] = useState<string | null>(null)

  // Form State
  const [deployName, setDeployName] = useState('My Web3Token')
  const [deploySymbol, setDeploySymbol] = useState('MW3')
  const [deployDecimals, setDeployDecimals] = useState('18')
  const [deployInitialSupply, setDeployInitialSupply] = useState('1000')
  const [interactTokenAddress, setInteractTokenAddress] = useState('')
  const [interactTo, setInteractTo] = useState('')
  const [interactAmount, setInteractAmount] = useState('10')
  const [interactTokenId, setInteractTokenId] = useState('0')

  // Context Menu State
  const [contextMenu, setContextMenu] = useState<ContextMenuData>(null)
  const contextMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const storedToken = localStorage.getItem('access_token')
    if (!storedToken) {
      navigate('/')
    } else {
      setToken(storedToken)
      fetchData(storedToken)
      fetchIdentities(storedToken)
    }
  }, [navigate])

  const fetchData = async (storedToken: string) => {
    try {
      const res = await fetch(`${gatewayUrl}/userinfo`, {
        headers: { 'Authorization': `Bearer ${storedToken}` }
      })
      if (res.ok) {
        const data = await res.json()
        setUserInfo(data)

        // Check for stored wallet address (set during SIWE login)
        const storedWallet = localStorage.getItem('siwe_wallet_address')
        if (storedWallet) {
          setWalletAddress(storedWallet)
        }

        if (data.sub) {
          const swRes = await fetch(`${gatewayUrl}/api/v1/wallet/address/${data.sub}`)
          if (swRes.ok) {
            const swData = await swRes.json()
            setSmartWalletAddress(swData.wallet_address)
            fetchPortfolio(swData.wallet_address)
          }
        }
      }
    } catch (err: any) {
      console.error(err)
    }
  }

  const fetchPortfolio = async (address: string) => {
    setPortfolioLoading(true)
    try {
      const res = await fetch(`http://localhost:3001/api/v2/addresses/${address}/token-balances`)
      if (res.ok) {
        const data = await res.json()
        const parsed: TokenBalance[] = data
          .filter((item: any) => item.token)
          .map((item: any) => ({
            address: item.token.address,
            decimals: Number(item.token.decimals || 0),
            symbol: item.token.symbol || '???',
            name: item.token.name || 'Unknown',
            balanceRaw: item.value,
            type: item.token.type,
            id: item.token_id || item.token.id || item.id,
          }))

        // Enhance with on-chain owner checks for mint dropdown filtering and verify token type
        const rpcUrl = (window as any).__RUNTIME_CONFIG__?.rpcUrl || 'http://localhost:8545'
        const provider = new ethers.JsonRpcProvider(rpcUrl)
        const enriched = await Promise.all(parsed.map(async (t) => {
          let isOwner = false;
          let correctedType = t.type;
          try {
            const contract = new ethers.Contract(t.address, [
              'function owner() view returns (address)', 
              'function hasRole(bytes32,address) view returns (bool)',
              'function supportsInterface(bytes4) view returns (bool)'
            ], provider)

            // Correct Blockscout caching issues on newly deployed tokens
            if (t.type === 'ERC-721' || t.type === 'ERC-1155') {
              try {
                const is1155 = await contract.supportsInterface('0xd9b67a26');
                if (is1155) correctedType = 'ERC-1155';
                else {
                  const is721 = await contract.supportsInterface('0x80ac58cd');
                  if (is721) correctedType = 'ERC-721';
                }
              } catch (e) { /* ignore if token doesn't support ERC165 */ }
            }

            const ownerAddr = await contract.owner()
            isOwner = ownerAddr.toLowerCase() === address.toLowerCase()
          } catch (e) {
            // Contract might not have owner(), network request failed, or is using roles API
          }
          return { ...t, isOwner, type: correctedType }
        }))

        setPortfolio(enriched)
      }
    } catch (err) {
      console.error("Failed to fetch portfolio:", err)
    } finally {
      setPortfolioLoading(false)
    }
  }

  useEffect(() => {
    if (activeSection === 'transfer' || activeSection === 'mint') {
      const filteredPortfolio = portfolio.filter(p => p.type.replace('-', '') === activeToken && (activeSection === 'mint' ? p.isOwner : true))
      const firstAsset = filteredPortfolio[0]
      if (firstAsset) {
        setSelectedAsset(firstAsset)
        setInteractTokenAddress(firstAsset.address)
        setInteractAmount('')
        if (firstAsset.id) {
          setInteractTokenId(firstAsset.id)
        }
      } else {
        setSelectedAsset(null)
        setInteractTokenAddress('')
        setInteractAmount('')
        setInteractTokenId(activeToken === 'ERC721' ? '' : '0')
      }
    }
  }, [activeToken, activeSection, portfolio])

  const fetchIdentities = async (storedToken: string) => {
    setIdentitiesLoading(true)
    try {
      const res = await fetch(`${gatewayUrl}/api/v1/accounts/me/identities`, {
        headers: { 'Authorization': `Bearer ${storedToken}` }
      })
      if (res.ok) {
        const data = await res.json()
        setLinkedIdentities(data.identities || [])
      }
    } catch (err) {
      console.error('Failed to fetch identities:', err)
    } finally {
      setIdentitiesLoading(false)
    }
  }

  const linkWallet = async () => {
    if (!(window as any).ethereum || !token) return
    setLinkingWallet(true)
    setLinkError(null)
    try {
      // 1. Request MetaMask account
      const accounts = await (window as any).ethereum.request({ method: 'eth_requestAccounts' })
      const address = accounts?.[0]?.toLowerCase()
      if (!address) { setLinkError('No wallet selected'); return }

      // 2. Get SIWE nonce
      const nonceRes = await fetch(`${gatewayUrl}/api/v1/siwe/nonce?address=${address}&protocol=siwe`)
      if (!nonceRes.ok) { setLinkError('Failed to get nonce'); return }
      const { message } = await nonceRes.json()

      // 3. Sign message
      const signature = await (window as any).ethereum.request({
        method: 'personal_sign',
        params: [message, address]
      })

      // 4. Link to account
      const linkRes = await fetch(`${gatewayUrl}/api/v1/auth/siwe/link`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ message, signature, protocol: 'siwe' })
      })

      if (!linkRes.ok) {
        const data = await linkRes.json()
        setLinkError(data.message || 'Linking failed')
        return
      }

      // 5. Refresh identities
      await fetchIdentities(token)
    } catch (err: any) {
      setLinkError(err.message || 'Linking cancelled')
    } finally {
      setLinkingWallet(false)
    }
  }

  const unlinkIdentity = async (identityId: string) => {
    if (!token || !confirm('Unlink this identity? The associated SCW assets will become inaccessible.')) return
    try {
      const res = await fetch(`${gatewayUrl}/api/v1/accounts/me/identities/${identityId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        setLinkError(data.message || 'Unlink failed')
        return
      }
      await fetchIdentities(token)
    } catch (err: any) {
      setLinkError(err.message || 'Unlink failed')
    }
  }

  const getProviderIcon = (provider: string) => {
    switch (provider) {
      case 'eoa': return '⟠'
      case 'google': return '🔵'
      case 'email': return '✉️'
      default: return '🔗'
    }
  }

  const executeTransaction = async (action: 'mint' | 'transfer' | 'deploy_contract', type: string, to?: string, amountOrId?: string, tokenAddr?: string, name?: string, symbol?: string, decimals?: string, initialSupply?: string) => {
    if (!userInfo?.sub) return;
    setTxLoading(true);
    setTxResult(null);
    setError(null);

    try {
      let finalAmount = amountOrId || '';
      if ((type === 'ERC20' || type === 'ERC1155') && finalAmount) {
        if (selectedAsset) {
          try {
            finalAmount = ethers.parseUnits(finalAmount, selectedAsset.decimals).toString();
          } catch (e: any) {
            throw new Error(`Invalid decimals in ${type} amount string: ` + e.message);
          }
        } else if (tokenAddr) {
          try {
            const r = await fetch(`http://localhost:3001/api/v2/tokens/${tokenAddr}`);
            let decimals = type === 'ERC20' ? 18 : 0;
            if (r.ok) {
              const data = await r.json();
              if (data.decimals !== undefined && data.decimals !== null) decimals = Number(data.decimals);
            }
            finalAmount = ethers.parseUnits(finalAmount, decimals).toString();
          } catch (e: any) {
            throw new Error(`Failed to parse ${type} amount: ` + e.message);
          }
        }
      }

      const res = await fetch(`${gatewayUrl}/api/v1/wallet/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          account_id: userInfo.sub,
          action: action,
          token_type: type,
          to: to || '',
          amount: finalAmount,
          token_id: interactTokenId || '',
          token_address: tokenAddr || '',
          name: name || '',
          symbol: symbol || '',
          decimals: decimals || '',
          initial_supply: initialSupply || ''
        })
      });

      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.message || 'Transaction failed');
      setTxResult(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setTxLoading(false);
    }
  }

  const copyText = (text: string) => {
    if (text) {
      navigator.clipboard.writeText(text)
    }
  }

  const truncateAddress = (addr: string) => `${addr.slice(0, 6)}...${addr.slice(-4)}`

  const blockscoutUrl = 'http://localhost:3001'

  const detectContextType = (value: string): ContextMenuType => {
    if (/^0x[a-fA-F0-9]{64}$/.test(value)) return 'tx'
    if (/^0x[a-fA-F0-9]{40}$/.test(value)) return 'address'
    return 'generic'
  }

  const handleContextMenu = (e: React.MouseEvent, value: string, forceType?: ContextMenuType) => {
    e.preventDefault()
    const type = forceType || detectContextType(value)
    setContextMenu({ x: e.clientX, y: e.clientY, value, type })
  }

  const handleFieldContextMenu = (e: React.MouseEvent, value: string) => {
    const type = detectContextType(value)
    if (type !== 'generic') {
      handleContextMenu(e, value, type)
    }
  }

  const getContextMenuItems = (type: ContextMenuType) => {
    switch (type) {
      case 'tx':
        return [
          { label: '🔍 Open in Blockscout', path: (v: string) => `/tx/${v}` },
          { label: '📋 View Internal Transactions', path: (v: string) => `/tx/${v}?tab=internal` },
          { label: '💱 View Token Transfers', path: (v: string) => `/tx/${v}?tab=token_transfers` },
          { label: '📜 View Logs', path: (v: string) => `/tx/${v}?tab=logs` },
          { label: '🔬 View Raw Trace', path: (v: string) => `/tx/${v}?tab=raw_trace` },
        ]
      case 'address':
        return [
          { label: '🔍 Open in Blockscout', path: (v: string) => `/address/${v}` },
          { label: '💰 View Token Holdings', path: (v: string) => `/address/${v}?tab=tokens` },
          { label: '📋 View Transactions', path: (v: string) => `/address/${v}?tab=txs` },
          { label: '🔗 View Internal Transactions', path: (v: string) => `/address/${v}?tab=internal_txns` },
          { label: '📜 View Logs', path: (v: string) => `/address/${v}?tab=logs` },
        ]
      default:
        return []
    }
  }

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null)
      }
    }
    if (contextMenu) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [contextMenu])

  const logout = async () => {
    // If wallet user, revoke MetaMask permissions
    if (walletAddress && (window as any).ethereum) {
      try {
        await (window as any).ethereum.request({
          method: 'wallet_revokePermissions',
          params: [{ eth_accounts: {} }]
        })
      } catch { /* some wallets don't support this */ }
    }
    localStorage.removeItem('access_token')
    const idToken = localStorage.getItem('id_token')
    localStorage.removeItem('id_token')
    localStorage.removeItem('loginChallenge')
    localStorage.removeItem('siwe_wallet_address')

    const redirectUrl = new URL(`${gatewayUrl}/oauth2/sessions/logout`)
    if (idToken) {
      redirectUrl.searchParams.append('id_token_hint', idToken)
      redirectUrl.searchParams.append('post_logout_redirect_uri', `${window.location.origin}/logout`)
    }
    window.location.href = redirectUrl.toString()
  }

  const copyAddress = (targetAddress?: string) => {
    const addr = typeof targetAddress === 'string' ? targetAddress : walletAddress;
    if (addr) {
      navigator.clipboard.writeText(addr)
      setAddressCopied(true)
      setTimeout(() => setAddressCopied(false), 2000)
    }
  }

  const copyToken = () => {
    if (token) {
      navigator.clipboard.writeText(token)
      setTokenCopied(true)
      setTimeout(() => setTokenCopied(false), 2000)
    }
  }

  const switchWallet = async () => {
    if (!(window as any).ethereum) {
      setSwitchStatus('MetaMask not found')
      return
    }
    try {
      setSwitchStatus('Requesting wallet...')
      const accounts = await (window as any).ethereum.request({ method: 'eth_requestAccounts' })
      const newAddress = accounts?.[0]?.toLowerCase()
      if (!newAddress) {
        setSwitchStatus('No wallet selected')
        return
      }
      if (newAddress === walletAddress?.toLowerCase()) {
        setSwitchStatus('Already connected to this wallet')
        setTimeout(() => setSwitchStatus(null), 2000)
        return
      }
      localStorage.removeItem('access_token')
      localStorage.removeItem('loginChallenge')
      localStorage.removeItem('siwe_wallet_address')
      window.location.href = '/'
    } catch {
      setSwitchStatus('Wallet request cancelled')
      setTimeout(() => setSwitchStatus(null), 2000)
    }
  }

  const isWalletUser = !!walletAddress && !userInfo?.email

  const renderSidebarItem = (title: string, section: MenuSection, icon: string, hasSubmenu = true) => {
    const isExpanded = activeSection === section;
    return (
      <div className="border-b border-gray-800">
        <button
          onClick={() => setActiveSection(section)}
          className={`w-full flex items-center justify-between px-6 py-4 text-sm font-semibold transition-colors
            ${isExpanded ? 'text-white bg-violet-600/10 border-l-2 border-violet-500' : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200 border-l-2 border-transparent'}
          `}
        >
          <span className="flex items-center gap-3">
            <span className="opacity-80 text-base">{icon}</span>
            <span className="tracking-wide uppercase text-xs">{title}</span>
          </span>
          {hasSubmenu ? (
            <svg className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180 text-violet-400' : 'text-gray-600'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
          ) : (
            isExpanded && <span className="w-1.5 h-1.5 rounded-full bg-violet-400"></span>
          )}
        </button>
        {hasSubmenu && isExpanded && (
          <div className="bg-[#0f172a] pb-3 pt-1">
            {(['ERC20', 'ERC721', 'ERC1155'] as TokenType[]).map(t => (
              <button
                key={t}
                onClick={() => setActiveToken(t)}
                className={`w-full text-left pl-14 pr-4 py-2 text-[13px] transition-all
                  ${activeToken === t
                    ? 'text-white font-medium bg-violet-600/20 shadow-[inset_2px_0_0_0_#3b82f6]'
                    : 'text-gray-500 hover:text-gray-300 hover:bg-white/5'}
                `}
              >
                {t} <span className="opacity-60 font-mono ml-1.5">- {t === 'ERC20' ? 'FT' : t === 'ERC721' ? 'NFT' : 'MULTI'}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    )
  }

  // Profile content for main area
  const renderProfileContent = () => (
    <div className="space-y-8">
      <header className="mb-10">
        <h2 className="text-3xl md:text-5xl font-extrabold text-white tracking-tight mb-4 flex items-center gap-4">
          Profile
          {isWalletUser && (
            <span className="px-3 py-1.5 bg-pink-500/10 border border-pink-500/20 text-pink-400 text-sm rounded-md font-mono mt-1">SIWE</span>
          )}
        </h2>
        <p className="text-gray-400 text-base md:text-lg max-w-2xl leading-relaxed">
          {isWalletUser
            ? 'Your Ethereum wallet identity and session details.'
            : 'Your account information and active session details.'}
        </p>
      </header>

      {/* User Card */}
      <div className={`rounded-2xl p-6 border backdrop-blur-md shadow-2xl ${
        isWalletUser
          ? 'bg-gradient-to-br from-pink-900/30 to-fuchsia-900/30 border-pink-500/20'
          : 'bg-gradient-to-br from-violet-900/30 to-fuchsia-900/30 border-violet-500/20'
      }`}>
        <div className="flex items-center gap-4">
          {userInfo?.picture ? (
            <img
              src={userInfo.picture}
              alt="Avatar"
              className="w-16 h-16 rounded-full border-2 border-white/20 shadow-lg"
            />
          ) : isWalletUser ? (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-pink-500 to-fuchsia-600 flex items-center justify-center text-white text-2xl font-bold shadow-lg shadow-pink-500/20">
              ⟠
            </div>
          ) : (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-600 flex items-center justify-center text-white text-xl font-bold shadow-lg shadow-violet-500/20">
              {(userInfo?.email?.[0] || userInfo?.sub?.[0] || '?').toUpperCase()}
            </div>
          )}
          <div className="min-w-0 flex-1">
            {isWalletUser ? (
              <>
                <h3 className="text-lg font-semibold text-white">Ethereum Wallet</h3>
                <div className="flex items-center gap-2 mt-0.5">
                  <p className="text-sm font-mono text-violet-300">{truncateAddress(walletAddress!)}</p>
                  <button
                    onClick={() => copyAddress()}
                    className="text-xs text-pink-400 hover:text-pink-300 font-medium transition-colors"
                    title="Copy full address"
                  >
                    {addressCopied ? '✅ Copied' : '📋 Copy'}
                  </button>
                </div>
                <span className="inline-flex items-center gap-1 mt-1 px-2 py-0.5 rounded-full text-xs font-medium bg-pink-500/20 text-pink-300 border border-pink-500/20">
                  ⟠ SIWE Connected
                </span>
              </>
            ) : (
              <>
                {userInfo?.name && (
                  <h3 className="text-lg font-semibold text-white truncate">{userInfo.name}</h3>
                )}
                {userInfo?.email && (
                  <p className="text-sm text-gray-400 truncate">{userInfo.email}</p>
                )}
                {userInfo?.email_verified !== undefined && (
                  <span className={`inline-flex items-center gap-1 mt-1 px-2 py-0.5 rounded-full text-xs font-medium border ${
                    userInfo.email_verified
                      ? 'bg-green-500/10 text-green-400 border-green-500/20'
                      : 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
                  }`}>
                    {userInfo.email_verified ? '✓ Verified' : '⚠ Unverified'}
                  </span>
                )}
              </>
            )}
          </div>
          <button
            onClick={logout}
            className="text-xs font-semibold px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-400 hover:text-red-300 rounded-lg transition-all border border-red-500/20"
          >
            {isWalletUser ? 'Disconnect & Sign Out' : 'Sign Out'}
          </button>
        </div>
      </div>

      {/* Wallet Details (SIWE users) */}
      {isWalletUser && (
        <div className="bg-[#1e293b]/70 border border-gray-700/50 shadow-2xl backdrop-blur-md rounded-2xl overflow-hidden">
          <div className="px-5 py-3.5 border-b border-gray-700/50 flex items-center justify-between">
            <h4 className="text-xs font-bold text-gray-400 uppercase tracking-widest">Wallet Details</h4>
          </div>
          <div className="divide-y divide-gray-700/30">
            <CopyableFieldDark label="address" value={walletAddress || ''} isMono onContextMenu={handleFieldContextMenu} />
            <CopyableFieldDark label="scw" value={smartWalletAddress || 'Deriving...'} isMono onContextMenu={handleFieldContextMenu} />
            <CopyableFieldDark label="auth" value="SIWE (EIP-4361)" />
            <CopyableFieldDark label="identity" value={userInfo?.identity_id || userInfo?.sub || ''} isMono />
          </div>
          <div className="px-5 py-3.5 border-t border-gray-700/50 flex items-center justify-between">
            <button
              onClick={switchWallet}
              className="text-sm text-pink-400 hover:text-pink-300 font-medium transition-colors flex items-center gap-2"
            >
              🔄 Switch Wallet
            </button>
            {switchStatus && (
              <span className="text-xs text-gray-500">{switchStatus}</span>
            )}
          </div>
        </div>
      )}

      {/* Account Details (non-wallet users) */}
      {userInfo && !isWalletUser && (
        <div className="bg-[#1e293b]/70 border border-gray-700/50 shadow-2xl backdrop-blur-md rounded-2xl overflow-hidden">
          <div className="px-5 py-3.5 border-b border-gray-700/50">
            <h4 className="text-xs font-bold text-gray-400 uppercase tracking-widest">Account Details</h4>
          </div>
          <div className="divide-y divide-gray-700/30">
            {Object.entries(userInfo)
              .filter(([key]) => !['picture', 'identity_id', 'sub'].includes(key))
              .map(([key, value]) => (
                <CopyableFieldDark key={key} label={key} value={value as string | boolean} />
              ))}
            <CopyableFieldDark label="scw" value={smartWalletAddress || 'Deriving...'} isMono onContextMenu={handleFieldContextMenu} />
            <CopyableFieldDark label="identity" value={userInfo?.identity_id || userInfo?.sub || ''} isMono />
          </div>
        </div>
      )}

      {/* Linked Identities & Wallets */}
      <div className="bg-[#1e293b]/70 border border-gray-700/50 shadow-2xl backdrop-blur-md rounded-2xl overflow-hidden">
        <div className="px-5 py-3.5 border-b border-gray-700/50 flex items-center justify-between">
          <h4 className="text-xs font-bold text-gray-400 uppercase tracking-widest">Linked Identities & Wallets</h4>
          <div className="flex items-center gap-2">
            <button
              onClick={linkWallet}
              disabled={linkingWallet}
              className="text-xs font-semibold px-3 py-1.5 bg-pink-500/10 hover:bg-pink-500/20 text-pink-400 hover:text-pink-300 rounded-lg transition-all border border-pink-500/20 disabled:opacity-50"
            >
              {linkingWallet ? '⏳ Linking...' : '⟠ Link Wallet'}
            </button>
          </div>
        </div>

        {linkError && (
          <div className="mx-5 mt-3 px-3 py-2 bg-red-900/30 border border-red-500/20 rounded-lg text-xs text-red-400">
            {linkError}
            <button onClick={() => setLinkError(null)} className="ml-2 text-red-500 hover:text-red-300">✕</button>
          </div>
        )}

        <div className="p-4 space-y-3">
          {identitiesLoading ? (
            <div className="text-center py-6">
              <div className="animate-spin inline-block rounded-full h-6 w-6 border-2 border-gray-700 border-t-violet-500"></div>
            </div>
          ) : linkedIdentities.length === 0 ? (
            <div className="text-center py-6 text-gray-500 text-sm">No linked identities found</div>
          ) : (
            linkedIdentities.map((ident) => (
              <div
                key={ident.identity_id}
                className="bg-[#0b1120]/60 border border-gray-700/40 rounded-xl p-4 flex items-center gap-4 hover:border-gray-600/50 transition-colors"
              >
                <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-gray-700 to-gray-800 flex items-center justify-center text-lg flex-shrink-0">
                  {getProviderIcon(ident.provider_id)}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-white capitalize">{ident.provider_id === 'eoa' ? 'Ethereum Wallet' : ident.provider_id}</span>
                    {ident.is_primary && (
                      <span className="px-1.5 py-0.5 bg-violet-500/20 text-violet-400 text-[10px] font-bold rounded border border-violet-500/20">PRIMARY</span>
                    )}
                  </div>
                  <p className="text-xs font-mono text-violet-300 truncate mt-0.5">{ident.provider_user_id}</p>
                  {ident.scw_address && (
                    <p className="text-[10px] font-mono text-gray-500 mt-1">SCW: {ident.scw_address.slice(0, 10)}...{ident.scw_address.slice(-8)}</p>
                  )}
                </div>
                {!ident.is_primary && (
                  <button
                    onClick={() => unlinkIdentity(ident.identity_id)}
                    className="text-xs text-red-400/60 hover:text-red-400 transition-colors flex-shrink-0"
                    title="Unlink this identity"
                  >
                    ✕
                  </button>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Access Token */}
      <div className="bg-[#1e293b]/70 border border-gray-700/50 shadow-2xl backdrop-blur-md rounded-2xl overflow-hidden">
        <div className="px-5 py-3.5 border-b border-gray-700/50 flex items-center justify-between">
          <button
            onClick={() => setShowToken(!showToken)}
            className="text-xs font-bold text-gray-400 uppercase tracking-widest flex items-center gap-2 hover:text-gray-300 transition-colors"
          >
            Access Token
            <svg className={`w-3.5 h-3.5 transition-transform ${showToken ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
          </button>
          <button
            onClick={copyToken}
            className={`text-xs font-medium transition-colors ${tokenCopied ? 'text-green-400' : 'text-violet-400 hover:text-violet-300'}`}
          >
            {tokenCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        {showToken && (
          <div className="p-4">
            <pre className="text-xs font-mono text-violet-300/80 break-all whitespace-pre-wrap bg-[#0b1120] rounded-lg p-4 max-h-32 overflow-y-auto border border-gray-700/50">
              {token}
            </pre>
          </div>
        )}
      </div>
    </div>
  )

  if (!userInfo) {
    return (
      <div className="h-screen w-full bg-[#0b1120] flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-4 border-gray-800 border-t-violet-500"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[#0b1120] text-gray-300 font-sans flex flex-col md:flex-row overflow-hidden selection:bg-violet-500/30">

      {/* 📱 Mobile Topbar */}
      <div className="md:hidden flex items-center justify-between p-4 bg-[#0f172a] border-b border-gray-800">
        <h1 className="text-xl font-bold tracking-tight text-white">Console</h1>
        <button onClick={() => logout()} className="text-xs font-semibold px-3 py-1.5 bg-gray-800 hover:bg-red-900/50 hover:text-red-400 rounded-md transition-colors">Logout</button>
      </div>

      {/* 🖥️ Sidebar (Fixed left) */}
      <div className="hidden md:flex w-72 flex-col bg-[#0f172a] border-r border-gray-800 h-screen flex-shrink-0 z-10 shadow-xl">
        {/* Brand Area */}
        <div className="p-6">
           <div className="flex items-center gap-3 mb-6">
             <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-600 flex items-center justify-center text-white font-bold shadow-lg shadow-violet-500/20">W</div>
             <h1 className="text-xl font-bold tracking-tight text-white">Developer Console</h1>
           </div>

           <div className="mt-8 bg-[#1e293b] rounded-lg p-3.5 border border-gray-700/50 shadow-inner">
             <div className="text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1.5 flex items-center gap-1.5">
               <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span> ERC-4337 Wallet
             </div>
             <div className="flex items-center justify-between group">
               <span className="font-mono text-xs text-violet-300">
                 {smartWalletAddress ? truncateAddress(smartWalletAddress) : 'Deriving...'}
               </span>
               {smartWalletAddress && (
                  <button
                    onClick={() => {
                      copyText(smartWalletAddress)
                      setScwCopied(true)
                      setTimeout(() => setScwCopied(false), 2000)
                    }}
                    onContextMenu={(e) => handleContextMenu(e, smartWalletAddress, 'address')}
                    className={`transition-all ${scwCopied ? 'text-green-400 opacity-100' : 'text-gray-500 hover:text-white opacity-0 group-hover:opacity-100'}`}
                    title="Copy · Right-click for more"
                  >
                    {scwCopied ? (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7"></path></svg>
                    ) : (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2 2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                    )}
                  </button>
                )}
             </div>
           </div>
        </div>

        {/* Navigation Elements */}
        <div className="flex-1 overflow-y-auto pb-6 custom-scrollbar mt-2">
          <div className="px-6 py-2 text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Toolchain</div>
          {renderSidebarItem('Deploy Contract', 'deploy', '🚀')}
          {renderSidebarItem('Mint Asset', 'mint', '✨')}
          {renderSidebarItem('Transfer Asset', 'transfer', '💸')}

          <div className="px-6 py-2 mt-4 text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1">Account</div>
          {renderSidebarItem('Profile', 'profile', '👤', false)}
        </div>

        {/* User Footer */}
        <div className="p-2 border-t border-gray-800 bg-[#0b1120]/50 backdrop-blur-sm">
           <button
             onClick={() => logout()}
             className="w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium text-red-400/80 hover:text-red-300 hover:bg-red-500/10 transition-all"
           >
             <span className="text-base">🚪</span>
             Sign Out
           </button>
        </div>
      </div>

      {/* 🚀 Main Content Area (Scrollable right) */}
      <div className="flex-1 overflow-y-auto h-screen relative">
        {/* Subtle Background Elements */}
        <div className="absolute top-0 inset-x-0 h-[400px] bg-gradient-to-b from-violet-900/10 to-transparent pointer-events-none"></div>
        <div className="absolute top-[-20%] left-[-10%] w-[50%] h-[50%] rounded-full bg-violet-500/5 blur-[120px] pointer-events-none"></div>

        <div className="p-6 md:p-12 lg:p-16 max-w-5xl mx-auto relative z-10 space-y-8">

          {error && (
            <div className="bg-red-900/40 border border-red-500/30 text-red-300 text-sm px-5 py-3.5 rounded-lg flex items-center gap-3 backdrop-blur-md">
              <svg className="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
              {error}
            </div>
          )}

          {/* Profile View */}
          {activeSection === 'profile' && renderProfileContent()}

          {/* Toolchain Views */}
          {activeSection !== 'profile' && (
            <>
              {/* Header */}
              <header className="mb-10">
                <h2 className="text-3xl md:text-5xl font-extrabold text-white tracking-tight mb-4 flex items-center gap-4">
                  {activeSection === 'deploy' ? 'Deploy Smart Contract' : activeSection === 'mint' ? 'Mint / Originate Asset' : 'Transfer Portfolio'}
                  <span className="px-3 py-1.5 bg-violet-500/10 border border-violet-500/20 text-violet-400 text-sm rounded-md font-mono mt-1">{activeToken}</span>
                </h2>
                <p className="text-gray-400 text-base md:text-lg max-w-2xl leading-relaxed">
                  {activeSection === 'deploy' ? 'Construct a new ERC factory and deploy a programmable token onto the blockchain securely via Web3Lab primitives.' :
                   activeSection === 'mint' ? 'Generate new tokens and assign cryptography ownership to your Abstract Smart Contract Wallet seamlessly.' :
                   'Transfer existing tokens to any Web3 wallet natively using Gasless Paymaster Sponsorship.'}
                </p>
              </header>

              {/* Form Container */}
              <div className="bg-[#1e293b]/70 border border-gray-700/50 shadow-2xl backdrop-blur-md rounded-2xl p-6 md:p-8">

                {/* Deploy View */}
                {activeSection === 'deploy' && (
                  <div className="space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Token Name</label>
                        <input
                          type="text"
                          placeholder="e.g. Web3Lab Token"
                          value={deployName}
                          onChange={(e) => setDeployName(e.target.value)}
                          className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                          disabled={txLoading}
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Token Symbol</label>
                        <input
                          type="text"
                          placeholder="e.g. W3L"
                          value={deploySymbol}
                          onChange={(e) => setDeploySymbol(e.target.value)}
                          className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                          disabled={txLoading}
                        />
                      </div>

                      {activeToken === 'ERC20' && (
                        <>
                          <div className="space-y-2">
                            <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                              Mint Amount (Initial Supply)
                              <span className="text-[9px] bg-violet-500/20 text-violet-400 px-2 py-0.5 rounded border border-violet-500/30">CUSTOM</span>
                            </label>
                            <input
                              type="text"
                              placeholder="e.g. 1000000"
                              value={deployInitialSupply}
                              onChange={(e) => setDeployInitialSupply(e.target.value)}
                              className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                              disabled={txLoading}
                            />
                          </div>
                          <div className="space-y-2">
                            <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                              Decimals (Precision)
                              <span className="text-[9px] bg-violet-500/20 text-violet-400 px-2 py-0.5 rounded border border-violet-500/30">CUSTOM</span>
                            </label>
                            <input
                              type="text"
                              placeholder="e.g. 18"
                              value={deployDecimals}
                              onChange={(e) => setDeployDecimals(e.target.value)}
                              className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                              disabled={txLoading}
                            />
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                )}

                {/* Interact View */}
                {(activeSection === 'mint' || activeSection === 'transfer') && (
                  <div className="space-y-6">
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Select Asset (Portfolio)</label>
                        {portfolioLoading ? (
                          <div className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-gray-500 text-sm">Loading portfolio from blockchain...</div>
                        ) : (
                          <div className="relative">
                            <select
                              value={portfolio.filter(p => p.type.replace('-', '') === activeToken && (activeSection === 'mint' ? p.isOwner : true)).some(p => p.address === interactTokenAddress && (p.id || '') === (interactTokenId === '0' && !p.id ? '' : interactTokenId)) ? `${interactTokenAddress}_${interactTokenId}` : 'custom'}
                              onChange={(e) => {
                                if (e.target.value === 'custom') {
                                  setSelectedAsset(null);
                                  setInteractTokenAddress('');
                                  setInteractAmount('');
                                  setAmountError(null);
                                } else {
                                  const [addr, idStr] = e.target.value.split('_');
                                  const asset = portfolio.find(p => p.address === addr && (p.id || '') === (idStr || ''));
                                  if (asset) {
                                    setSelectedAsset(asset);
                                    setInteractTokenAddress(asset.address);
                                    setInteractAmount('');
                                    setAmountError(null);
                                    if (asset.id !== undefined) {
                                      setInteractTokenId(asset.id);
                                    }
                                  }
                                }
                              }}
                              className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white outline-none transition-all shadow-inner focus:border-violet-500 appearance-none pr-10"
                              disabled={txLoading}
                            >
                              <option value="" disabled>Select a {activeToken} token to {activeSection}...</option>
                              {portfolio.filter(p => p.type.replace('-', '') === activeToken && (activeSection === 'mint' ? p.isOwner : true)).map(asset => {
                                const formattedBalance = ethers.formatUnits(asset.balanceRaw, asset.decimals);
                                const displayBalance = formattedBalance.length > 10 ? Number(formattedBalance).toFixed(4) : formattedBalance;
                                return (
                                  <option key={`${asset.address}_${asset.id || ''}`} value={`${asset.address}_${asset.id || ''}`}>
                                    {asset.name} ({asset.symbol}) {asset.id ? `| ID: ${asset.id} ` : ''}— Bal: {displayBalance} {activeSection === 'mint' && asset.isOwner && '⭐ Owned'}
                                  </option>
                                )
                              })}
                              <option value="custom">+(Enter Custom Contract Address)</option>
                            </select>
                            <div className="absolute inset-y-0 right-0 flex items-center px-4 pointer-events-none text-gray-500">
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
                            </div>
                          </div>
                        )}
                      </div>
                      {(!interactTokenAddress || !portfolio.filter(p => p.type.replace('-', '') === activeToken && (activeSection === 'mint' ? p.isOwner : true)).some(p => p.address === interactTokenAddress)) && (
                        <div className="space-y-2 animate-fade-in">
                          <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Target Token Address (0x)</label>
                          <input
                            type="text"
                            placeholder={`Enter the deployed ${activeToken} contract address`}
                            value={interactTokenAddress}
                            onChange={(e) => {
                              setInteractTokenAddress(e.target.value);
                              setSelectedAsset(null);
                            }}
                            className="w-full px-5 py-3.5 font-mono text-sm bg-[#0b1120] border border-gray-700 rounded-xl text-violet-300 placeholder-gray-700 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                            disabled={txLoading}
                          />
                        </div>
                      )}
                    </div>

                    <div className="space-y-4">
                      <div className="space-y-2">
                        <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">
                          {activeSection === 'mint' ? 'Mint To Address (0x)' : 'Recipient Address (0x)'}
                        </label>
                        <div className="relative">
                          <select
                            value={linkedIdentities.some(id => id.scw_address === interactTo) ? interactTo : (interactTo === '' ? 'self' : 'custom')}
                            onChange={(e) => {
                              if (e.target.value === 'custom') {
                                setInteractTo('custom');
                              } else if (e.target.value === 'self') {
                                setInteractTo('');
                              } else {
                                setInteractTo(e.target.value);
                              }
                            }}
                            className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white outline-none transition-all shadow-inner focus:border-violet-500 appearance-none pr-10"
                            disabled={txLoading}
                          >
                            <option value="self">Self (Current SCW Address)</option>
                            {linkedIdentities.filter(id => id.scw_address && id.scw_address !== smartWalletAddress).map(ident => (
                              <option key={ident.identity_id} value={ident.scw_address}>
                                {ident.provider_id === 'eoa' ? 'Ethereum Wallet' : ident.provider_id} ({ident.provider_user_id}) - {ident.scw_address?.slice(0, 6)}...{ident.scw_address?.slice(-4)}
                              </option>
                            ))}
                            <option value="custom">+(Enter Custom Destination Address)</option>
                          </select>
                          <div className="absolute inset-y-0 right-0 flex items-center px-4 pointer-events-none text-gray-500">
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
                          </div>
                        </div>
                      </div>

                      {(!linkedIdentities.some(id => id.scw_address === interactTo) && interactTo !== '') && (
                        <div className="space-y-2 animate-fade-in">
                          <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Target Identity Address (0x)</label>
                          <input
                            type="text"
                            placeholder="Destination address to receive the tokens"
                            value={interactTo === 'custom' ? '' : interactTo}
                            onChange={(e) => setInteractTo(e.target.value)}
                            className="w-full px-5 py-3.5 font-mono text-sm bg-[#0b1120] border border-gray-700 rounded-xl text-violet-300 placeholder-gray-700 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner"
                            disabled={txLoading}
                          />
                        </div>
                      )}
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      {activeToken !== 'ERC721' && (
                        <div className="space-y-2 relative">
                          <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                            Amount (Tokens)
                            {activeSection === 'transfer' && selectedAsset && (
                              <span className="text-[9px] bg-fuchsia-500/20 text-fuchsia-400 px-2 py-0.5 rounded border border-fuchsia-500/30">
                                Decimals: {selectedAsset.decimals}
                              </span>
                            )}
                          </label>
                          <div className="relative">
                            <input
                              type="text"
                              placeholder={
                                activeSection === 'transfer' && selectedAsset
                                  ? `Max: ${ethers.formatUnits(selectedAsset.balanceRaw, selectedAsset.decimals)}`
                                  : "e.g. 10.5"
                              }
                              value={interactAmount}
                              onChange={(e) => {
                                const val = e.target.value;
                                setInteractAmount(val);
                                setAmountError(null);
                                if (activeSection === 'transfer' && selectedAsset && val) {
                                  const parts = val.split('.');
                                  if (parts.length > 1 && parts[1].length > selectedAsset.decimals) {
                                    setAmountError(`Warning: Only ${selectedAsset.decimals} decimal places allowed.`);
                                  }
                                }
                              }}
                              className={`w-full px-5 py-3.5 bg-[#0b1120] border ${amountError ? 'border-red-500 focus:border-red-500 focus:ring-red-500' : 'border-gray-700 focus:border-violet-500 focus:ring-violet-500'} rounded-xl text-white placeholder-gray-700 focus:ring-1 outline-none transition-all shadow-inner ${activeSection === 'transfer' && selectedAsset ? 'pr-20' : ''}`}
                              disabled={txLoading || (activeSection === 'transfer' && !selectedAsset)}
                            />
                            {activeSection === 'transfer' && selectedAsset && (
                              <button
                                onClick={() => {
                                  setInteractAmount(ethers.formatUnits(selectedAsset.balanceRaw, selectedAsset.decimals));
                                  setAmountError(null);
                                }}
                                className="absolute right-3 top-1/2 -translate-y-1/2 px-3 py-1 bg-violet-500/10 text-violet-400 hover:text-violet-300 hover:bg-violet-500/20 text-xs font-bold rounded-lg border border-violet-500/20 transition-colors shadow-sm"
                                disabled={txLoading}
                              >
                                MAX
                              </button>
                            )}
                          </div>
                          {amountError && <p className="text-red-400 text-xs mt-1 pl-1 absolute -bottom-5 left-0">{amountError}</p>}
                        </div>
                      )}
                      {activeToken !== 'ERC20' && (
                        <div className="space-y-2">
                          <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                            Token Identifier (ID)
                            {(activeSection === 'mint' && activeToken === 'ERC721') && (
                              <span className="text-[9px] bg-[#0b1120] text-gray-400 px-2 py-0.5 rounded border border-gray-700">AUTO-INCREMENT</span>
                            )}
                          </label>
                          <input
                            type="text"
                            placeholder="e.g. 1"
                            value={(activeSection === 'mint' && activeToken === 'ERC721') ? "Auto-assigned by Contract" : interactTokenId}
                            onChange={(e) => setInteractTokenId(e.target.value)}
                            className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-700 focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all shadow-inner disabled:opacity-50 disabled:cursor-not-allowed"
                            disabled={txLoading || (activeSection === 'mint' && activeToken === 'ERC721')}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Submit Action */}
                <div className="pt-8 mt-2">
                  <button
                    onClick={() => {
                      if (activeSection === 'deploy') {
                        executeTransaction('deploy_contract', activeToken, '', '', '', deployName, deploySymbol, deployDecimals, deployInitialSupply)
                      } else {
                        executeTransaction(activeSection as 'mint'|'transfer', activeToken, interactTo === 'custom' ? '' : interactTo, interactAmount, interactTokenAddress)
                      }
                    }}
                    disabled={!smartWalletAddress || txLoading || (activeSection === 'deploy' ? (!deployName || !deploySymbol) : (!interactTokenAddress || interactTokenAddress === 'custom' || interactTo === 'custom'))}
                    className={`w-full py-4 px-6 rounded-xl font-bold text-white shadow-lg transition-all border border-black/10 flex items-center justify-center
                      ${(!smartWalletAddress || (activeSection === 'deploy' ? (!deployName || !deploySymbol) : (!interactTokenAddress || interactTokenAddress === 'custom' || interactTo === 'custom')))
                        ? 'bg-gray-800 cursor-not-allowed text-gray-500 shadow-none'
                        : txLoading
                          ? 'bg-violet-600/50 cursor-wait'
                          : 'bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:from-violet-500 hover:to-fuchsia-500 transform hover:-translate-y-0.5 hover:shadow-violet-500/25'}
                    `}
                  >
                    {txLoading ? (
                      <div className="flex items-center gap-3">
                        <svg className="animate-spin h-5 w-5 text-white" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                        Processing Transaction via Paymaster...
                      </div>
                    ) : activeSection === 'deploy' ? (
                      <>Compile & Deploy {activeToken} Smart Contract</>
                    ) : activeSection === 'mint' ? (
                      <>Execute Cryptographic Mint (100% Gasless)</>
                    ) : (
                      <>Execute Secure Transfer (100% Gasless)</>
                    )}
                  </button>
                </div>
              </div>

              {/* TX Terminal Style Results */}
              {txResult && (
                <div className="bg-[#0b1120]/80 backdrop-blur-xl border border-violet-500/30 rounded-2xl p-6 shadow-2xl relative overflow-hidden">
                  <div className="absolute top-0 left-0 w-1 h-full bg-gradient-to-b from-violet-400 to-fuchsia-500"></div>

                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3 text-violet-400 font-bold tracking-wide">
                      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                      {txResult.message || 'TRANSACTION SUCCESSFUL'}
                    </div>
                    <div className="text-[10px] uppercase font-mono tracking-widest text-green-400 bg-green-400/10 px-3 py-1 rounded-full border border-green-400/20">Bundled via 4337</div>
                  </div>

                  <div className="grid grid-cols-1 gap-4">
                    <div className="bg-[#1e293b]/50 border border-gray-700/50 rounded-xl p-4" onContextMenu={(e) => handleContextMenu(e, txResult.transaction_hash, 'tx')}>
                      <span className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-2 font-mono">Blockchain TX Hash</span>
                      <div className="flex items-center gap-3">
                        <p className="flex-1 text-sm font-mono text-violet-300 truncate">{txResult.transaction_hash}</p>
                        <button
                          onClick={() => {
                            copyText(txResult.transaction_hash)
                            setTxCopied(true)
                            setTimeout(() => setTxCopied(false), 2000)
                          }}
                          onContextMenu={(e) => handleContextMenu(e, txResult.transaction_hash, 'tx')}
                          className={`flex-shrink-0 p-2 rounded-lg shadow-sm transition-all border ${
                            txCopied
                              ? 'text-green-400 bg-green-400/10 border-green-500/50'
                              : 'text-gray-400 hover:text-white bg-[#0b1120] border-gray-700 hover:border-violet-500/50'
                          }`}
                          title="Copy TX Hash · Right-click for more"
                        >
                          {txCopied ? (
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7"></path></svg>
                          ) : (
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2 2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                          )}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {/* Global Right-click Context Menu */}
          {contextMenu && contextMenu.type !== 'generic' && (
            <div
              ref={contextMenuRef}
              className="fixed z-50 min-w-[240px] bg-[#1e293b] border border-gray-600/50 rounded-xl shadow-2xl shadow-black/50 backdrop-blur-xl overflow-hidden"
              style={{ left: contextMenu.x, top: contextMenu.y }}
            >
              <div className="px-3 py-2 border-b border-gray-700/50">
                <span className="text-[9px] font-bold text-gray-500 uppercase tracking-widest">
                  {contextMenu.type === 'tx' ? 'Transaction Actions' : 'Address Actions'}
                </span>
              </div>
              {getContextMenuItems(contextMenu.type).map((item, i) => (
                <button
                  key={i}
                  onClick={() => {
                    window.open(`${blockscoutUrl}${item.path(contextMenu.value)}`, '_blank')
                    setContextMenu(null)
                  }}
                  className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:bg-violet-600/20 hover:text-white transition-colors flex items-center gap-2"
                >
                  {item.label}
                </button>
              ))}
              <div className="border-t border-gray-700/50">
                <button
                  onClick={() => {
                    copyText(contextMenu.value)
                    setContextMenu(null)
                  }}
                  className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:bg-violet-600/20 hover:text-white transition-colors flex items-center gap-2"
                >
                  📎 Copy {contextMenu.type === 'tx' ? 'TX Hash' : 'Address'}
                </button>
              </div>
            </div>
          )}

        </div>
      </div>
    </div>
  )
}
