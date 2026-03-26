import { useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'

type MenuSection = 'profile' | 'deploy' | 'mint' | 'transfer';
type TokenType = 'ERC20' | 'ERC721' | 'ERC1155';

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
      <span className={`text-sm text-gray-300 break-all flex-1 ${isMono ? 'font-mono text-amber-300' : ''}`}>
        {stringValue}
      </span>
      <span className={`text-xs font-medium transition-opacity flex-shrink-0 ${copied ? 'text-green-400 opacity-100' : 'text-amber-400 opacity-0 group-hover:opacity-100'}`}>
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

  // Layout State
  const [activeSection, setActiveSection] = useState<MenuSection>('deploy')
  const [activeToken, setActiveToken] = useState<TokenType>('ERC20')

  // Profile State
  const [switchStatus, setSwitchStatus] = useState<string | null>(null)
  const [tokenCopied, setTokenCopied] = useState(false)
  const [addressCopied, setAddressCopied] = useState(false)
  const [showToken, setShowToken] = useState(false)

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
          }
        }
      }
    } catch (err: any) {
      console.error(err)
    }
  }

  const executeTransaction = async (action: 'mint' | 'transfer' | 'deploy_contract', type: string, to?: string, amountOrId?: string, tokenAddr?: string, name?: string, symbol?: string, decimals?: string, initialSupply?: string) => {
    if (!userInfo?.sub) return;
    setTxLoading(true);
    setTxResult(null);
    setError(null);

    try {
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
          amount: amountOrId || '',
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
            ${isExpanded ? 'text-white bg-amber-600/10 border-l-2 border-amber-500' : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200 border-l-2 border-transparent'}
          `}
        >
          <span className="flex items-center gap-3">
            <span className="opacity-80 text-base">{icon}</span>
            <span className="tracking-wide uppercase text-xs">{title}</span>
          </span>
          {hasSubmenu ? (
            <svg className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180 text-amber-400' : 'text-gray-600'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
          ) : (
            isExpanded && <span className="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
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
                    ? 'text-white font-medium bg-amber-600/20 shadow-[inset_2px_0_0_0_#f59e0b]'
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
            <span className="px-3 py-1.5 bg-yellow-500/10 border border-yellow-500/20 text-yellow-400 text-sm rounded-md font-mono mt-1">SIWE</span>
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
          ? 'bg-gradient-to-br from-yellow-900/30 to-orange-900/30 border-yellow-500/20'
          : 'bg-gradient-to-br from-amber-900/30 to-orange-900/30 border-amber-500/20'
      }`}>
        <div className="flex items-center gap-4">
          {userInfo?.picture ? (
            <img
              src={userInfo.picture}
              alt="Avatar"
              className="w-16 h-16 rounded-full border-2 border-white/20 shadow-lg"
            />
          ) : isWalletUser ? (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-yellow-500 to-orange-600 flex items-center justify-center text-white text-2xl font-bold shadow-lg shadow-yellow-500/20">
              ⟠
            </div>
          ) : (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center text-white text-xl font-bold shadow-lg shadow-amber-500/20">
              {(userInfo?.email?.[0] || userInfo?.sub?.[0] || '?').toUpperCase()}
            </div>
          )}
          <div className="min-w-0 flex-1">
            {isWalletUser ? (
              <>
                <h3 className="text-lg font-semibold text-white">Ethereum Wallet</h3>
                <div className="flex items-center gap-2 mt-0.5">
                  <p className="text-sm font-mono text-amber-300">{truncateAddress(walletAddress!)}</p>
                  <button
                    onClick={() => copyAddress()}
                    className="text-xs text-yellow-400 hover:text-yellow-300 font-medium transition-colors"
                    title="Copy full address"
                  >
                    {addressCopied ? '✅ Copied' : '📋 Copy'}
                  </button>
                </div>
                <span className="inline-flex items-center gap-1 mt-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500/20 text-yellow-300 border border-yellow-500/20">
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
              className="text-sm text-yellow-400 hover:text-yellow-300 font-medium transition-colors flex items-center gap-2"
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
            className={`text-xs font-medium transition-colors ${tokenCopied ? 'text-green-400' : 'text-amber-400 hover:text-amber-300'}`}
          >
            {tokenCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        {showToken && (
          <div className="p-4">
            <pre className="text-xs font-mono text-amber-300/80 break-all whitespace-pre-wrap bg-[#0b1120] rounded-lg p-4 max-h-32 overflow-y-auto border border-gray-700/50">
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
        <div className="animate-spin rounded-full h-12 w-12 border-4 border-gray-800 border-t-amber-500"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[#0b1120] text-gray-300 font-sans flex flex-col md:flex-row overflow-hidden selection:bg-amber-500/30">

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
             <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center text-white font-bold shadow-lg shadow-amber-500/20">W</div>
             <h1 className="text-xl font-bold tracking-tight text-white">Developer Console</h1>
           </div>

           <div className="mt-8 bg-[#1e293b] rounded-lg p-3.5 border border-gray-700/50 shadow-inner">
             <div className="text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-1.5 flex items-center gap-1.5">
               <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span> ERC-4337 Wallet
             </div>
             <div className="flex items-center justify-between group">
               <span className="font-mono text-xs text-amber-300">
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
        <div className="absolute top-0 inset-x-0 h-[400px] bg-gradient-to-b from-amber-900/10 to-transparent pointer-events-none"></div>
        <div className="absolute top-[-20%] left-[-10%] w-[50%] h-[50%] rounded-full bg-amber-500/5 blur-[120px] pointer-events-none"></div>

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
                  <span className="px-3 py-1.5 bg-amber-500/10 border border-amber-500/20 text-amber-400 text-sm rounded-md font-mono mt-1">{activeToken}</span>
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
                          className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
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
                          className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
                          disabled={txLoading}
                        />
                      </div>

                      {activeToken === 'ERC20' && (
                        <>
                          <div className="space-y-2">
                            <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                              Mint Amount (Initial Supply)
                              <span className="text-[9px] bg-amber-500/20 text-amber-400 px-2 py-0.5 rounded border border-amber-500/30">CUSTOM</span>
                            </label>
                            <input
                              type="text"
                              placeholder="e.g. 1000000"
                              value={deployInitialSupply}
                              onChange={(e) => setDeployInitialSupply(e.target.value)}
                              className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
                              disabled={txLoading}
                            />
                          </div>
                          <div className="space-y-2">
                            <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1 flex items-center justify-between">
                              Decimals (Precision)
                              <span className="text-[9px] bg-amber-500/20 text-amber-400 px-2 py-0.5 rounded border border-amber-500/30">CUSTOM</span>
                            </label>
                            <input
                              type="text"
                              placeholder="e.g. 18"
                              value={deployDecimals}
                              onChange={(e) => setDeployDecimals(e.target.value)}
                              className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-600 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
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
                    <div className="space-y-2">
                      <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Target Token Address (0x)</label>
                      <input
                        type="text"
                        placeholder="Enter the deployed ERC token contract address"
                        value={interactTokenAddress}
                        onChange={(e) => setInteractTokenAddress(e.target.value)}
                        className="w-full px-5 py-3.5 font-mono text-sm bg-[#0b1120] border border-gray-700 rounded-xl text-amber-300 placeholder-gray-700 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
                        disabled={txLoading}
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">
                        {activeSection === 'mint' ? 'Mint To Address (0x)' : 'Recipient Address (0x)'}
                      </label>
                      <input
                        type="text"
                        placeholder={activeSection === 'mint' ? "Leave empty to mint to yourself, or enter destination address" : "Destination address to receive the tokens"}
                        value={interactTo}
                        onChange={(e) => setInteractTo(e.target.value)}
                        className="w-full px-5 py-3.5 font-mono text-sm bg-[#0b1120] border border-gray-700 rounded-xl text-amber-300 placeholder-gray-700 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
                        disabled={txLoading}
                      />
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      {activeToken !== 'ERC721' && (
                        <div className="space-y-2">
                          <label className="text-xs font-bold text-gray-400 uppercase tracking-widest pl-1">Amount</label>
                          <input
                            type="text"
                            placeholder="e.g. 1000"
                            value={interactAmount}
                            onChange={(e) => setInteractAmount(e.target.value)}
                            className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-700 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner"
                            disabled={txLoading}
                          />
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
                            className="w-full px-5 py-3.5 bg-[#0b1120] border border-gray-700 rounded-xl text-white placeholder-gray-700 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none transition-all shadow-inner disabled:opacity-50 disabled:cursor-not-allowed"
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
                        executeTransaction(activeSection as 'mint'|'transfer', activeToken, interactTo, interactAmount, interactTokenAddress)
                      }
                    }}
                    disabled={!smartWalletAddress || txLoading || (activeSection === 'deploy' ? (!deployName || !deploySymbol) : !interactTokenAddress)}
                    className={`w-full py-4 px-6 rounded-xl font-bold text-white shadow-lg transition-all border border-black/10 flex items-center justify-center
                      ${(!smartWalletAddress || (activeSection === 'deploy' ? (!deployName || !deploySymbol) : !interactTokenAddress))
                        ? 'bg-gray-800 cursor-not-allowed text-gray-500 shadow-none'
                        : txLoading
                          ? 'bg-amber-600/50 cursor-wait'
                          : 'bg-gradient-to-r from-amber-600 to-orange-600 hover:from-amber-500 hover:to-orange-500 transform hover:-translate-y-0.5 hover:shadow-amber-500/25'}
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
                <div className="bg-[#0b1120]/80 backdrop-blur-xl border border-amber-500/30 rounded-2xl p-6 shadow-2xl relative overflow-hidden">
                  <div className="absolute top-0 left-0 w-1 h-full bg-gradient-to-b from-amber-400 to-orange-500"></div>

                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3 text-amber-400 font-bold tracking-wide">
                      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                      {txResult.message || 'TRANSACTION SUCCESSFUL'}
                    </div>
                    <div className="text-[10px] uppercase font-mono tracking-widest text-green-400 bg-green-400/10 px-3 py-1 rounded-full border border-green-400/20">Bundled via 4337</div>
                  </div>

                  <div className="grid grid-cols-1 gap-4">
                    <div className="bg-[#1e293b]/50 border border-gray-700/50 rounded-xl p-4" onContextMenu={(e) => handleContextMenu(e, txResult.transaction_hash, 'tx')}>
                      <span className="block text-[10px] font-bold text-gray-500 uppercase tracking-widest mb-2 font-mono">Blockchain TX Hash</span>
                      <div className="flex items-center gap-3">
                        <p className="flex-1 text-sm font-mono text-amber-300 truncate">{txResult.transaction_hash}</p>
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
                              : 'text-gray-400 hover:text-white bg-[#0b1120] border-gray-700 hover:border-amber-500/50'
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
                  className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:bg-amber-600/20 hover:text-white transition-colors flex items-center gap-2"
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
                  className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:bg-amber-600/20 hover:text-white transition-colors flex items-center gap-2"
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
