import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

interface UserInfo {
  sub: string
  email?: string
  name?: string
  picture?: string
  email_verified?: boolean
  [key: string]: any
}

const CopyableField = ({ label, value, isMono = false }: { label: string, value: string | boolean, isMono?: boolean }) => {
  const [copied, setCopied] = useState(false)
  const stringValue = typeof value === 'boolean' ? (value ? 'true' : 'false') : String(value)
  
  const handleCopy = () => {
    navigator.clipboard.writeText(stringValue)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div 
      className="px-4 py-2.5 flex items-start gap-3 hover:bg-gray-50 transition-colors group cursor-pointer" 
      onClick={handleCopy}
      title="Click to copy"
    >
      <span className="text-xs font-mono text-gray-400 min-w-[100px] pt-0.5">{label}</span>
      <span className={`text-sm text-gray-800 break-all flex-1 ${isMono ? 'font-mono' : ''}`}>
        {stringValue}
      </span>
      <span className={`text-xs font-medium transition-opacity ${copied ? 'text-green-600 opacity-100' : 'text-amber-600 opacity-0 group-hover:opacity-100'}`}>
        {copied ? 'Copied!' : 'Copy'}
      </span>
    </div>
  )
}

export const ProfilePage = () => {
  const navigate = useNavigate()
  const { gatewayUrl } = (window as any).__RUNTIME_CONFIG__ || { gatewayUrl: 'https://gateway.web3-local-dev.com' }
  const [token, setToken] = useState<string | null>(null)
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [walletAddress, setWalletAddress] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [switchStatus, setSwitchStatus] = useState<string | null>(null)
  const [tokenCopied, setTokenCopied] = useState(false)
  const [addressCopied, setAddressCopied] = useState(false)

  useEffect(() => {
    const storedToken = localStorage.getItem('access_token')
    if (!storedToken) {
      navigate('/')
      return
    }
    setToken(storedToken)

    const fetchData = async () => {
      try {
        // Fetch userinfo from Hydra
        const res = await fetch(`${gatewayUrl}/userinfo`, {
          headers: { 'Authorization': `Bearer ${storedToken}` }
        })
        if (!res.ok) throw new Error(`Failed to fetch user info (${res.status})`)
        const data = await res.json()
        setUserInfo(data)

        // Check for stored wallet address (set during SIWE login)
        const storedWallet = localStorage.getItem('siwe_wallet_address')
        if (storedWallet) {
          setWalletAddress(storedWallet)
        }
      } catch (err: any) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [navigate, gatewayUrl])

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
    
    // OIDC RP-Initiated Logout parameters
    const redirectUrl = new URL(`${gatewayUrl}/oauth2/sessions/logout`)
    if (idToken) {
      redirectUrl.searchParams.append('id_token_hint', idToken)
      redirectUrl.searchParams.append('post_logout_redirect_uri', `${window.location.origin}/logout`)
    }
    window.location.href = redirectUrl.toString()
  }

  const copyToken = () => {
    if (token) {
      navigator.clipboard.writeText(token)
      setTokenCopied(true)
      setTimeout(() => setTokenCopied(false), 2000)
    }
  }

  const copyAddress = (targetAddress?: string) => {
    const addr = typeof targetAddress === 'string' ? targetAddress : walletAddress;
    if (addr) {
      navigator.clipboard.writeText(addr)
      setAddressCopied(true)
      setTimeout(() => setAddressCopied(false), 2000)
    }
  }

  const truncateAddress = (addr: string) =>
    `${addr.slice(0, 6)}...${addr.slice(-4)}`

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
      // Different wallet — clear session and re-login
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

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="animate-spin rounded-full h-10 w-10 border-4 border-amber-200 border-t-amber-600 mb-4"></div>
        <p className="text-gray-500 text-sm">Loading profile...</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between border-b pb-4">
        <h2 className="text-2xl font-bold text-gray-900">Profile</h2>
        <button
          onClick={logout}
          className="text-sm text-red-500 hover:text-red-700 font-medium transition-colors"
        >
          {isWalletUser ? 'Disconnect & Sign Out' : 'Sign Out'}
        </button>
      </div>

      {/* User Card */}
      <div className={`rounded-xl p-6 border shadow-sm ${
        isWalletUser
          ? 'bg-gradient-to-br from-yellow-50 to-orange-50 border-yellow-100'
          : 'bg-gradient-to-br from-amber-50 to-orange-50 border-amber-100'
      }`}>
        <div className="flex items-center gap-4">
          {userInfo?.picture ? (
            <img
              src={userInfo.picture}
              alt="Avatar"
              className="w-16 h-16 rounded-full border-2 border-white shadow-md"
            />
          ) : isWalletUser ? (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-yellow-500 to-orange-600 flex items-center justify-center text-white text-2xl font-bold shadow-md">
              ⟠
            </div>
          ) : (
            <div className="w-16 h-16 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center text-white text-xl font-bold shadow-md">
              {(userInfo?.email?.[0] || userInfo?.sub?.[0] || '?').toUpperCase()}
            </div>
          )}
          <div className="min-w-0 flex-1">
            {isWalletUser ? (
              <>
                <h3 className="text-lg font-semibold text-gray-900">Ethereum Wallet</h3>
                <div className="flex items-center gap-2 mt-0.5">
                  <p className="text-sm font-mono text-gray-600">{truncateAddress(walletAddress)}</p>
                  <button
                    onClick={() => copyAddress()}
                    className="text-xs text-yellow-600 hover:text-yellow-800 font-medium transition-colors"
                    title="Copy full address"
                  >
                    {addressCopied ? '✅ Copied' : '📋 Copy'}
                  </button>
                </div>
                <span className="inline-flex items-center gap-1 mt-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                  ⟠ SIWE Connected
                </span>
              </>
            ) : (
              <>
                {userInfo?.name && (
                  <h3 className="text-lg font-semibold text-gray-900 truncate">{userInfo.name}</h3>
                )}
                {userInfo?.email && (
                  <p className="text-sm text-gray-600 truncate">{userInfo.email}</p>
                )}
                {userInfo?.email_verified !== undefined && (
                  <span className={`inline-flex items-center gap-1 mt-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                    userInfo.email_verified
                      ? 'bg-green-100 text-green-800'
                      : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {userInfo.email_verified ? '✓ Verified' : '⚠ Unverified'}
                  </span>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Wallet Details (for SIWE users) */}
      {isWalletUser && (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <div className="px-4 py-3 bg-gray-50 border-b border-gray-200">
            <h4 className="text-sm font-semibold text-gray-700">Wallet Details</h4>
          </div>
          <div className="divide-y divide-gray-100">
            <CopyableField label="address" value={walletAddress || ''} isMono />
            <CopyableField label="auth_method" value="SIWE (EIP-4361)" />
            <CopyableField label="identity_id" value={userInfo?.identity_id || userInfo?.sub || ''} isMono />
          </div>
          <div className="px-4 py-3 border-t border-gray-100 flex items-center justify-between">
            <button
              onClick={switchWallet}
              className="text-sm text-yellow-600 hover:text-yellow-800 font-medium transition-colors"
            >
              🔄 Switch Wallet
            </button>
            {switchStatus && (
              <span className="text-xs text-gray-500">{switchStatus}</span>
            )}
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="bg-red-50 text-red-700 text-sm px-4 py-3 rounded-lg border border-red-200">
          <strong>Error:</strong> {error}
             {/* User Details (for non-wallet users) */}
      {userInfo && !isWalletUser && (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden shadow-sm">
          <div className="px-4 py-3 bg-gray-50 border-b border-gray-200">
            <h4 className="text-sm font-semibold text-gray-700">Account Details</h4>
          </div>
          <div className="divide-y divide-gray-100">
            {Object.entries(userInfo)
              .filter(([key]) => !['picture'].includes(key))
              .map(([key, value]) => (
                <CopyableField key={key} label={key} value={value} />
              ))}
          </div>
        </div>
      )}       </div>
      )}

      {/* Access Token */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
          <h4 className="text-sm font-semibold text-gray-700">Access Token</h4>
          <button
            onClick={copyToken}
            className={`text-xs font-medium transition-colors ${tokenCopied ? 'text-green-600' : 'text-amber-600 hover:text-amber-800'}`}
          >
            {tokenCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        <div className="p-4">
          <pre className="text-xs font-mono text-gray-600 break-all whitespace-pre-wrap bg-gray-50 rounded p-3 max-h-24 overflow-y-auto">
            {token}
          </pre>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-3 pt-2">
        <button
          onClick={() => navigate('/')}
          className="flex-1 bg-gray-100 hover:bg-gray-200 text-gray-700 font-medium py-2.5 rounded-lg transition-colors text-sm"
        >
          Home
        </button>
        <button
          onClick={() => navigate('/dashboard')}
          className="flex-1 bg-amber-50 hover:bg-amber-100 text-amber-700 font-medium py-2.5 rounded-lg transition-colors text-sm"
        >
          Dashboard
        </button>
      </div>
    </div>
  )
}
