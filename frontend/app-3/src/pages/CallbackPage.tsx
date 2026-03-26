import { useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'

export const CallbackPage = () => {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const code = searchParams.get('code')
  const [status, setStatus] = useState('Exchanging authorization code...')

  useEffect(() => {
    if (!code) {
      setStatus('Error: No authorization code found in URL.')
      return
    }

    const exchangeCode = async () => {
      try {
        const { gatewayUrl, clientId, authDomain } = (window as any).__RUNTIME_CONFIG__
        
        // Exchange the code using URLSearchParams for x-www-form-urlencoded
        const body = new URLSearchParams()
        body.append('grant_type', 'authorization_code')
        body.append('code', code)
        body.append('redirect_uri', `https://${authDomain}/callback`)
        body.append('client_id', clientId)

        const res = await fetch(`${gatewayUrl}/oauth2/token`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body
        })

        const data = await res.json()
        
        if (data.access_token) {
          localStorage.setItem('access_token', data.access_token)
          if (data.id_token) {
            localStorage.setItem('id_token', data.id_token)
          }
          // Clear login challenge state
          localStorage.removeItem('loginChallenge')
          navigate('/profile')
        } else {
          setStatus(`Failed to get access token: ${JSON.stringify(data)}`)
        }
      } catch (err: any) {
        setStatus(`Error during token exchange: ${err.message}`)
      }
    }

    exchangeCode()
  }, [code, navigate])

  return (
    <div className="text-center py-8">
      <h2 className="text-xl font-bold mb-4">Authenticating...</h2>
      <p className="text-gray-500 text-sm animate-pulse">{status}</p>
    </div>
  )
}
