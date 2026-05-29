# OpsOwl AI Data Handling Policy

**Version:** 1.0  
**Effective Date:** 2026-06-01  
**Owner:** Engineering & Compliance  
**Classification:** Internal — Customer-Facing on Request

---

## 1. Purpose

This policy defines how OpsOwl processes customer data through its AI subsystem. It establishes two formal deployment modes, specifies exactly what data is transmitted to AI services in each mode, sets disclosure obligations, and provides configuration guidance for regulated deployments.

---

## 2. AI Deployment Modes

OpsOwl ships with two mutually exclusive AI deployment modes. The mode is selected at installation time and is visible on the Settings page of every deployment.

### Mode A — Cloud AI

| Attribute | Value |
|-----------|-------|
| **Config** | `DEFAULT_PROVIDER=anthropic` or `DEFAULT_PROVIDER=openai` |
| **Inference runs on** | Provider's cloud infrastructure (Anthropic / OpenAI / Azure OpenAI) |
| **Data leaves customer network** | YES — document content is transmitted over TLS to provider APIs |
| **Suitable for** | Internal fintech startups, non-regulated enterprises, environments where DPA is in place with provider |
| **Not suitable for** | RBI-regulated NBFCs, banks, insurance companies, AML teams handling PII-dense documents without DPA |

### Mode B — Air-Gapped AI (Recommended for Regulated Clients)

| Attribute | Value |
|-----------|-------|
| **Config** | `DEFAULT_PROVIDER=ollama` |
| **Inference runs on** | Customer's own servers (Ollama runtime, `llama3.1:8b` by default) |
| **Data leaves customer network** | NO — all inference, embeddings, and RAG retrieval stay within the customer's infrastructure |
| **Suitable for** | All regulated deployments — NBFCs, UCBs, mid-sized banks, AML teams, government entities |
| **Not suitable for** | Customers whose hardware cannot run a 7–13B parameter LLM (minimum: 16 GB RAM, modern CPU or any NVIDIA GPU) |

**Default at installation:** Mode A (Cloud AI) is the default for developer/evaluation builds. Production deployments for regulated clients MUST explicitly set `DEFAULT_PROVIDER=ollama` during setup.

---

## 3. What Data Is Processed by AI

The following table covers every AI operation and the exact data payload sent to the inference engine.

| Feature | Data Sent to AI | PII Masking Applied? | Notes |
|---------|----------------|----------------------|-------|
| **AI Copilot Q&A** | User's question + retrieved document chunk text (RAG) | Yes — on the question only | Document chunks are NOT masked before embedding lookup; masked question is used for prompt |
| **Case Risk Analysis** | Up to 40 document chunk texts for the case | No | Full chunk content including any text extracted from uploaded PDFs/images |
| **Comms Entity Extraction** | Full body of the selected communication (email/SMS transcript) | No | Raw communication text is sent as-is |
| **STR/Report Summary** | Case title, description, status, assignee name, document count | No | No document content; only structured case metadata |
| **Embeddings (RAG indexing)** | Full text of every uploaded document, chunked | No | Processed once on upload; stored as vectors in pgvector |

### What PII Masking Covers

The masking layer (`pii_masker.py`) intercepts the copilot question text and replaces the following patterns before the prompt is sent to any LLM:

- **PAN card numbers** (format: AAAAA9999A)
- **Aadhaar numbers** (12-digit)
- **Indian mobile numbers** (10-digit, starting 6–9)
- **Email addresses**
- **IFSC codes**
- **Bank account numbers** (9–18 digit sequences)

**What masking does NOT cover:** business names, transaction narratives, amounts, free-text descriptions, document body text, and any unstructured data that does not match the patterns above.

**Implication:** For Mode A (Cloud AI), document content sent during Risk Analysis and Comms Entity Extraction is transmitted to external providers without masking. Customers who use Cloud AI mode accept that case document content may be processed on provider infrastructure.

---

## 4. Approved Cloud AI Providers (Mode A Only)

Only the following providers may be configured when using Cloud AI mode:

| Provider | Acceptable Agreement | Data Residency |
|----------|---------------------|----------------|
| **Anthropic** (Claude) | Anthropic Commercial Terms + DPA | US-based; no Indian region |
| **OpenAI** | OpenAI Enterprise Agreement + DPA | US-based; no Indian region |
| **Azure OpenAI** | Microsoft Enterprise Agreement + DPA | India-Central region available — **preferred for Mode A in India** |

Using Azure OpenAI with `eastus` or `centralindia` region endpoints is the recommended cloud option for Indian clients who need data residency but cannot run on-premise AI.

Any other third-party AI provider requires explicit approval from OpsOwl Engineering and a signed DPA review before being added to configuration.

---

## 5. Customer Disclosure Obligations

### OpsOwl's Obligation to Customers

Before go-live, OpsOwl must disclose in writing:

1. Which AI mode the deployment uses (Cloud AI or Air-Gapped AI).
2. If Cloud AI: the name of the provider, the DPA reference number, and the data residency region.
3. That document content is processed by the AI subsystem during risk analysis and copilot queries.
4. That PII masking is limited in scope (Section 3 above).

This disclosure is delivered via the **Deployment Agreement** or **Order Form** addendum. Template wording is in `docs/legal/AI_DISCLOSURE_ADDENDUM.md` (to be created).

### Customer's Obligation to Their Regulators

Regulated customers (NBFCs, UCBs, banks) must obtain their own regulatory clearance for any AI processing of case documents. OpsOwl recommends Air-Gapped Mode for all RBI-regulated clients and will document this recommendation in the deployment agreement.

---

## 6. Data Retention by AI Subsystem

| Data Type | Where Stored | Retention |
|-----------|-------------|-----------|
| Document embeddings (vectors) | Customer's PostgreSQL (pgvector) | Deleted when document is deleted |
| AI Copilot responses | `ai_logs` table in customer DB | 90 days default; configurable |
| Risk analysis output | `case_insights` table in customer DB | Permanent (part of case record) |
| LLM prompt/response logs | `ai_logs` table | 90 days default |
| PII token maps | In-memory only, per request | Discarded at end of HTTP request |

**No AI request data is retained by OpsOwl after transmission.** All AI logs live in the customer's database on their infrastructure.

For Cloud AI providers: Anthropic and OpenAI do not retain API inputs beyond their standard API terms (zero retention for enterprise agreements). Customers on Azure OpenAI must configure zero-retention per their Azure contract.

---

## 7. Regulatory Alignment

### RBI Data Localisation

RBI's 2018 circular on storage of payment system data and subsequent NBFC IT Framework circulars require that all data collected in India be stored in India. This applies to case documents processed by OpsOwl.

**Recommendation:** All RBI-regulated clients must use Air-Gapped Mode (Mode B). If a regulated client insists on Cloud AI, only Azure OpenAI with `centralindia` region is acceptable, and a specific RBI data localisation opinion from the client's legal team is required before OpsOwl enables Cloud AI for that deployment.

### SEBI / IRDAI / FIU-IND

The same recommendation applies. Air-Gapped Mode eliminates the data residency question entirely.

---

## 8. Security Controls

| Control | Implementation |
|---------|---------------|
| **Transport encryption** | All API calls to cloud providers use TLS 1.2+ (enforced by provider SDKs) |
| **No prompt logging to disk** | Prompts are assembled in memory and sent directly; never written to disk |
| **AI API key storage** | Keys stored as env vars; never in DB or code; rotated via deployment update |
| **Prompt injection detection** | `copilot.py` contains injection pattern detection; suspicious queries are rejected |
| **Access control** | AI endpoints require valid JWT; Copilot is rate-limited per user |

---

## 9. AI Mode Configuration Reference

### Setting Mode B (Air-Gapped) at Deployment

In `apps/ai-service/.env` (or Docker Compose override):

```env
DEFAULT_PROVIDER=ollama
OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=llama3.1:8b
OLLAMA_EMBEDDING_MODEL=nomic-embed-text
```

In Docker Compose, add the Ollama service:

```yaml
ollama:
  image: ollama/ollama:latest
  volumes:
    - ollama_models:/root/.ollama
  ports:
    - "11434:11434"
  deploy:
    resources:
      reservations:
        devices:
          - capabilities: [gpu]   # remove if no GPU
```

Pull models before first run:

```bash
docker exec opsowl-ollama ollama pull llama3.1:8b
docker exec opsowl-ollama ollama pull nomic-embed-text
```

### Setting Mode A (Cloud AI) at Deployment

In `apps/ai-service/.env`:

```env
DEFAULT_PROVIDER=anthropic     # or: openai, azure_openai
ANTHROPIC_API_KEY=sk-ant-...
# OR
OPENAI_API_KEY=sk-...
```

### Verifying Active Mode (Admin UI)

The active AI provider is shown in **Settings → AI Configuration**. Any administrator can confirm the current mode without accessing the server. A green "Air-Gapped" badge or orange "Cloud AI: [Provider]" badge appears at the top of the AI Configuration card.

---

## 10. Incident Response

If a data handling incident occurs involving the AI subsystem:

1. **Immediate:** Disable AI endpoints via `AI_SERVICE_ENABLED=false` in the API env and restart.
2. **Within 1 hour:** Identify which AI mode was active and what data was in-flight.
3. **Within 24 hours:** For Cloud AI incidents, contact the provider's DPA/privacy team and reference your agreement.
4. **For Mode B (Air-Gapped):** Data does not leave the network; incident scope is limited to the customer's own infrastructure.
5. **Notification:** Follow the customer's incident response policy; OpsOwl will provide technical assistance on request.

---

## 11. Policy Review

This policy is reviewed annually or when:
- A new AI provider or model is added
- RBI / SEBI issues new circulars affecting AI data processing
- A material change is made to what data the AI subsystem processes

**Document owner:** Engineering Lead  
**Last reviewed:** 2026-05-29  
**Next review due:** 2027-05-29
