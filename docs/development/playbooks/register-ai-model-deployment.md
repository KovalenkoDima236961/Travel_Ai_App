# Register AI Model Deployment

1. Verify the model and adapter records exist in `ai_models` and
   `ai_model_adapters`.
2. Confirm adapter checksum and approval status.
3. Create a deployment payload with `status='registered'`,
   `trafficMode='disabled'`, `internalOnly=true`, `promptVersion`, and a human
   `reason`.
4. Run `scripts/ai/register-model-deployment.sh --payload deployment.json`.
   Trip Service writes the `ai_model_deployments` row and
   `ai_model_deployment_events.action='created'` in one transaction.
5. Do not enable traffic during registration.
