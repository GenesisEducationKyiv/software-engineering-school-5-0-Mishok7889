package notification

// WeatherUpdateTemplate is the HTML template for weather notification emails
const WeatherUpdateTemplate = `
		<h2>Weather Update for {{.City}}</h2>
		<div style="background-color: #f5f5f5; padding: 20px; border-radius: 8px; margin: 20px 0;">
			<h3 style="color: #333; margin-top: 0;">Current Weather</h3>
			<p style="font-size: 18px; margin: 10px 0;">
				<strong>Temperature:</strong> {{.Temperature}}{{.TemperatureUnit}}
			</p>
			<p style="font-size: 16px; margin: 10px 0;">
				<strong>Humidity:</strong> {{.Humidity}}{{.HumidityUnit}}
			</p>
			<p style="font-size: 16px; margin: 10px 0;">
				<strong>Description:</strong> {{.Description}}
			</p>
			<p style="font-size: 14px; color: #666; margin: 10px 0;">
				<strong>Last Updated:</strong> {{.LastUpdated}}
			</p>
		</div>
		<hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
		<p style="font-size: 12px; color: #888;">
			You are receiving this because you subscribed to <strong>{{.Frequency}}</strong> weather updates for <strong>{{.SubscriptionCity}}</strong>.
		</p>
		{{if .UnsubscribeURL}}
		<p style="font-size: 12px; color: #888;">
			To unsubscribe from these updates, <a href="{{.UnsubscribeURL}}" style="color: #0066cc;">click here</a>.
		</p>
		{{end}}
`
