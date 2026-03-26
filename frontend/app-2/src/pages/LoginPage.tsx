import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { BrowserProvider } from 'ethers'

export const LoginPage = () => {
  const [searchParams] = useSearchParams()
  const { gatewayUrl } = (window as any).__RUNTIME_CONFIG__
  const loginChallenge = searchParams.get('login_challenge') || localStorage.getItem('loginChallenge')
  const [status, setStatus] = useState<string>('Initializing...')

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [siweLoading, setSiweLoading] = useState(false)

  useEffect(() => {
    if (loginChallenge) {
      localStorage.setItem('loginChallenge', loginChallenge)
      // Check if user already has a Kratos session via /sessions/whoami.
      // Do NOT call self-service/login/browser?login_challenge=xxx here —
      // that binds the Hydra challenge to a Kratos flow, making it unusable
      // for the SIWE backend's direct AcceptLoginRequest.
      fetch(`${gatewayUrl}/identity/sessions/whoami`, {
        headers: { 'Accept': 'application/json' },
        credentials: 'include'
      }).then(res => {
        if (res.ok) {
          // User already has a Kratos session → complete OAuth2 flow
          window.location.href = '/'
        } else {
          setStatus('Ready for login')
        }
      }).catch(() => {
        setStatus('Ready for login')
      })
    } else {
      setStatus('Waiting for login challenge...')
    }
  }, [loginChallenge])

  const completeHydraLogin = () => {
    setStatus('Completing login...')
    // Redirect to app root to start a fresh OAuth2 flow.
    // Since the user has a Kratos session, Hydra will auto-accept (skip=true)
    // and the flow completes instantly without showing the login page again.
    window.location.href = '/'
  }

  // --- SIWE FLOW ---
  const handleSiweLogin = async () => {
    if (siweLoading) return
    setSiweLoading(true)
    try {
      if (!(window as any).ethereum) throw new Error('MetaMask or Web3 wallet not found')
      setStatus('Connecting to wallet...')
      const provider = new BrowserProvider((window as any).ethereum)
      await provider.send('eth_requestAccounts', [])
      const signer = await provider.getSigner()
      const address = await signer.getAddress()

      setStatus('Fetching SIWE challenge...')
      // 1. Fetch nonce + preformatted message from backend
      const nonceRes = await fetch(`${gatewayUrl}/api/v1/siwe/nonce?address=${address}&protocol=eip712`)
      if (!nonceRes.ok) throw new Error('Failed to fetch nonce')
      const nonceData = await nonceRes.json()

      setStatus('Please sign the message in your wallet...')
      // 2. Sign the EIP-712 message from backend
      const signature = await provider.send('eth_signTypedData_v4', [address, nonceData.message])

      setStatus('Verifying signature & completing login...')
      // 3. Authenticate: verify signature + complete Hydra OAuth2 flow
      const authRes = await fetch(`${gatewayUrl}/api/v1/siwe/authenticate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          message: nonceData.message,
          signature,
          protocol: 'eip712',
          login_challenge: loginChallenge
        })
      })

      if (!authRes.ok) {
        const errData = await authRes.json().catch(() => ({}))
        throw new Error(errData.message || 'Verification failed')
      }

      const authData = await authRes.json()

      // 4. Store wallet address for profile page (MetaMask loses connection after redirects)
      localStorage.setItem('siwe_wallet_address', address)

      // 5. Follow Hydra redirect
      if (authData.redirect_to) {
        window.location.href = authData.redirect_to
      } else {
        completeHydraLogin()
      }
    } catch (e: any) {
      console.error(e)
      const msg = e.message || ''
      // If login challenge is expired/consumed, clear it and get a fresh one
      if (msg.includes('expired') || msg.includes('unauthorized')) {
        localStorage.removeItem('loginChallenge')
        setStatus('Login challenge expired, getting a fresh one...')
        setTimeout(() => { window.location.href = '/' }, 1500)
        return
      }
      setStatus(`SIWE Error: ${msg}`)
    } finally {
      setSiweLoading(false)
    }
  }

  // --- GOOGLE FLOW (Simulated redirection for Kratos OIDC) ---
  const handleGoogleLogin = async (e: FormEvent) => {
    e.preventDefault()
    setStatus('Initiating Google Login...')
    
    try {
      // 1. Fetch browser flow
      const query = loginChallenge ? `?login_challenge=${loginChallenge}` : ''
      const res = await fetch(`${gatewayUrl}/identity/self-service/login/browser${query}`, {
        headers: { 'Accept': 'application/json' },
        credentials: 'include'
      })
      const flow = await res.json()
      
      // Find OIDC traits (Google)
      const actionUrl = flow.ui.action
      const csrfToken = flow.ui.nodes.find((n: any) => n.attributes.name === 'csrf_token')?.attributes.value

      if (!actionUrl || !csrfToken) {
        throw new Error("Could not initialize OIDC flow from Kratos.")
      }

      // Submit standard form to trigger Kratos redirection out of the SPA
      const form = document.createElement('form')
      form.method = 'POST'
      form.action = actionUrl

      const csrfInput = document.createElement('input')
      csrfInput.type = 'hidden'
      csrfInput.name = 'csrf_token'
      csrfInput.value = csrfToken

      const providerInput = document.createElement('input')
      providerInput.type = 'hidden'
      providerInput.name = 'provider'
      providerInput.value = 'google'

      form.appendChild(csrfInput)
      form.appendChild(providerInput)
      document.body.appendChild(form)
      form.submit()
      
    } catch (e: any) {
      setStatus(`Google Flow Error: ${e.message}`)
    }
  }

  // --- EMAIL SIGN IN FLOW ---
  const handleEmailSignIn = async (e: FormEvent) => {
    e.preventDefault()
    if (!email || !password) return setStatus('Please enter email and password')
    setStatus('Initiating Email Sign In...')
    
    try {
      const query = loginChallenge ? `?login_challenge=${loginChallenge}` : ''
      const res = await fetch(`${gatewayUrl}/identity/self-service/login/browser${query}`, {
        headers: { 'Accept': 'application/json' },
        credentials: 'include'
      })
      const flow = await res.json()

      // If user already has a session, Kratos may return redirect instead of a login flow
      const flowRedirect = flow.redirect_browser_to
        || flow.continue_with?.find((c: any) => c.action === 'redirect_browser_to')?.redirect_browser_to
      if (flowRedirect) {
        window.location.href = flowRedirect
        return
      }

      if (!flow.ui?.action) {
        // No flow UI and no redirect — user may already be logged in
        completeHydraLogin()
        return
      }

      const actionUrl = flow.ui.action
      const csrfToken = flow.ui.nodes.find((n: any) => n.attributes.name === 'csrf_token')?.attributes.value

      setStatus('Submitting credentials...')
      const submitRes = await fetch(actionUrl, {
        method: 'POST',
        headers: { 'Accept': 'application/json', 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          method: 'password',
          identifier: email,
          password: password,
          csrf_token: csrfToken
        })
      })

      const submitData = await submitRes.json()

      // Kratos with oauth2_provider returns 422 + redirect_browser_to on SUCCESS
      const redirectUrl = submitData.redirect_browser_to
        || submitData.continue_with?.find((c: any) => c.action === 'redirect_browser_to')?.redirect_browser_to
      if (redirectUrl) {
        window.location.href = redirectUrl
        return
      }

      if (!submitRes.ok) {
        throw new Error(submitData.ui?.messages?.[0]?.text || 'Invalid credentials')
      }

      completeHydraLogin()
    } catch (err: any) {
      setStatus(`Sign In Error: ${err.message}`)
    }
  }

  // --- EMAIL SIGN UP FLOW ---
  const handleEmailSignUp = async (e: FormEvent) => {
    e.preventDefault()
    if (!email || !password) return setStatus('Please enter email and password')
    setStatus('Initiating Email Sign Up...')
    
    try {
      // Create registration flow — Kratos inherits OAuth2 context from login flow cookies
      const res = await fetch(`${gatewayUrl}/identity/self-service/registration/browser`, {
        headers: { 'Accept': 'application/json' },
        credentials: 'include'
      })

      if (!res.ok) {
        // 400 typically means user already has a Kratos session — complete Hydra login with existing session
        if (res.status === 400) {
          setStatus('You already have an active session. Completing login...')
          completeHydraLogin()
          return
        }
        const errBody = await res.json().catch(() => ({}))
        throw new Error(errBody.error?.message || `Registration failed (${res.status})`)
      }

      const flow = await res.json()
      const actionUrl = flow.ui?.action
      if (!actionUrl) {
        throw new Error('Invalid registration flow')
      }
      const csrfToken = flow.ui.nodes.find((n: any) => n.attributes.name === 'csrf_token')?.attributes.value

      setStatus('Submitting registration...')
      const submitRes = await fetch(actionUrl, {
        method: 'POST',
        headers: { 'Accept': 'application/json', 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          method: 'password',
          traits: { email },
          password: password,
          csrf_token: csrfToken
        })
      })

      const submitData = await submitRes.json()

      // Kratos with oauth2_provider returns 422 + redirect_browser_to on SUCCESS
      // Check for redirect BEFORE checking submitRes.ok
      const redirectUrl = submitData.redirect_browser_to
        || submitData.continue_with?.find((c: any) => c.action === 'redirect_browser_to')?.redirect_browser_to
      if (redirectUrl) {
        window.location.href = redirectUrl
        return
      }

      if (!submitRes.ok) {
        const msg = submitData.ui?.messages?.[0]?.text
          || submitData.ui?.nodes?.find((n: any) => n.messages?.length > 0)?.messages?.[0]?.text
          || 'Registration failed'
        throw new Error(msg)
      }

      // Fallback if no redirect URL found
      completeHydraLogin()
    } catch (err: any) {
      setStatus(`Sign Up Error: ${err.message}`)
    }
  }

  return (
    <div className="flex flex-col space-y-4">
      <h2 className="text-2xl font-bold text-center mb-2">Welcome</h2>
      <div className="bg-emerald-50 text-emerald-800 text-xs px-3 py-2 rounded mb-4 text-center">
        {status}
      </div>

      <form className="flex flex-col space-y-3 p-4 border border-gray-200 rounded-lg bg-gray-50">
        <input 
          type="email" 
          placeholder="Email address" 
          value={email} 
          onChange={(e) => setEmail(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-emerald-500"
          required 
        />
        <input 
          type="password" 
          placeholder="Password" 
          value={password} 
          onChange={(e) => setPassword(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-emerald-500"
          required 
        />
        <div className="flex space-x-2 pt-2">
          <button disabled={!loginChallenge} onClick={handleEmailSignIn} className="flex-1 bg-emerald-600 text-white font-semibold py-2 rounded shadow hover:bg-emerald-700 disabled:opacity-50">
            Sign In
          </button>
          <button disabled={!loginChallenge} onClick={handleEmailSignUp} className="flex-1 bg-white border border-gray-300 text-gray-700 font-semibold py-2 rounded shadow hover:bg-gray-50 disabled:opacity-50">
            Sign Up
          </button>
        </div>
      </form>

      <div className="relative my-4">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-gray-300"></div>
        </div>
        <div className="relative flex justify-center text-sm">
          <span className="px-2 bg-white text-gray-500">Or continue with</span>
        </div>
      </div>

      <button disabled={!loginChallenge} onClick={handleGoogleLogin} className="w-full bg-white border border-gray-300 text-gray-700 font-semibold py-2.5 px-4 rounded shadow hover:bg-gray-50 flex justify-center items-center gap-2 disabled:opacity-50">
        <svg className="w-5 h-5" viewBox="0 0 24 24">
            <path fill="currentColor" d="M21.35,11.1H12.18V13.83H18.69C18.36,17.64 15.19,19.27 12.19,19.27C8.36,19.27 5,16.25 5,12C5,7.9 8.2,4.73 12.2,4.73C15.29,4.73 17.1,6.7 17.1,6.7L19,4.72C19,4.72 16.56,2 12.1,2C6.42,2 2.03,6.8 2.03,12C2.03,17.05 6.16,22 12.25,22C17.6,22 21.5,18.33 21.5,12.91C21.5,11.76 21.35,11.1 21.35,11.1V11.1Z" />
        </svg>
        Google
      </button>

      <button disabled={!loginChallenge || siweLoading} onClick={handleSiweLogin} className="w-full bg-gray-900 border border-gray-900 text-white font-semibold py-2.5 px-4 rounded shadow hover:bg-gray-800 disabled:opacity-50">
        Ethereum Wallet (SIWE)
      </button>

      {/* Development auto-resume after Google redirect */}
      <button onClick={completeHydraLogin} className="mt-6 text-xs text-gray-400 underline decoration-dashed w-full text-center">
        [Dev] Resume flow after Kratos session
      </button>

    </div>
  )
}
