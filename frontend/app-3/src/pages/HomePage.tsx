import { useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'

export const HomePage = () => {
  const [searchParams] = useSearchParams()
  const { gatewayUrl, clientId, authDomain } = (window as any).__RUNTIME_CONFIG__

  // Check if user just logged out
  const justLoggedOut = searchParams.get('logout') === 'true'

  const handleLogin = () => {
    // Clear logout flag and redirect to OAuth2 flow
    const redirectUri = encodeURIComponent(`https://${authDomain}/callback`)
    const state = crypto.randomUUID().replace(/-/g, '')
    const authUrl = `${gatewayUrl}/oauth2/auth?client_id=${clientId}&response_type=code&redirect_uri=${redirectUri}&scope=openid offline_access&state=${state}`
    window.location.href = authUrl
  }

  // Auto-trigger OAuth2 flow on mount ONLY if not just logged out
  useEffect(() => {
    if (!justLoggedOut) {
      handleLogin()
    }
  }, [])

  // If just logged out, show a manual sign-in button
  if (justLoggedOut) {
    return (
      <div className="text-center">
        <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight mb-1">Web3 Test Portal</h1>
        <p className="text-sm text-green-600 mb-6">You have been signed out successfully.</p>
        <button
          onClick={handleLogin}
          className="bg-amber-600 hover:bg-amber-700 text-white font-semibold py-3 px-8 rounded-lg transition-colors shadow-sm"
        >
          Sign In
        </button>
      </div>
    )
  }

  return (
    <div className="text-center">
      <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight mb-1">Web3 Test Portal</h1>
      <div className="flex items-center justify-center gap-2 mb-8">
        <p className="text-sm text-gray-500">Redirecting to login...</p>
      </div>
    </div>
  )
}
