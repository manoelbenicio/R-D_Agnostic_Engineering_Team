# PRD — Open Multi-Agent (OMA) — Architecture & Configuration Mappings

Este documento apresenta o mapeamento detalhado da arquitetura, fluxo de orquestração e padrões de configuração do framework **Open Multi-Agent (OMA)** (https://open-multi-agent.com), com foco especial na integração com os provedores **Anthropic** (modelos Opus) e **OpenAI**.

---

## 1. Visão Geral da Arquitetura

O **Open Multi-Agent (OMA)** (ou `@open-multi-agent/core`) é um framework de orquestração multi-agente nativo para TypeScript/Node.js baseado em uma filosofia **Goal-First** (baseada em objetivos), em oposição à abordagem tradicional **Graph-First** (onde o desenvolvedor precisa definir manualmente os nós e arestas de um grafo de estado).

### 1.1 Fluxo de Execução "Goal-to-DAG"
Quando a função principal `runTeam()` é invocada, o OMA executa um ciclo de vida automatizado em essa cinco etapas:

1. **Decomposição do Objetivo (Planning):** Um agente coordenador (`coordinator`) analisa o objetivo final do usuário (ex: *"Escreva uma API REST e execute testes de segurança"*). Ele divide a meta em tarefas discretas, infere as dependências sequenciais e paralelas entre elas, e gera um **DAG (Directed Acyclic Graph)** de tarefas em tempo real.
2. **Resolução de Dependências:** O motor de execução analisa o DAG e enfileira as tarefas na `TaskQueue`. As tarefas que não possuem dependências anteriores são marcadas como prontas para execução imediata.
3. **Execução Paralela (Fanning Out):** As tarefas sem dependência mútua são executadas de forma concorrente em múltiplos agentes (`AgentRunner` rodando em paralelo). Cada agente recebe apenas as ferramentas (`tools`) para as quais foi explicitamente autorizado (princípio de privilégio mínimo/default-deny).
4. **Comunicação em Tempo Real:** Conforme as tarefas são concluídas, os resultados intermediários são publicados em um barramento de mensagens compartilhado (`MessageBus`). Os agentes subsequentes no DAG consomem essas mensagens como inputs para suas respectivas tarefas.
5. **Síntese de Resultados:** Uma vez resolvidas todas as ramificações do DAG, o coordenador compila as saídas e gera um resultado final tipado e validado por esquemas (via `Zod`).

```mermaid
graph TD
    A[Objetivo do Usuário] --> B[Coordenador / Planner]
    B --> C[Geração de Grafo DAG de Tarefas]
    C --> D[Tarefa 1: Planejamento / Design]
    D --> E[Tarefa 2: Desenvolvimento CRUD]
    E --> F[Tarefa 3a: Geração de Testes - Paralelo]
    E --> G[Tarefa 4a: Revisão de Código - Paralelo]
    F --> H[Tarefa Final: Síntese e Validação]
    G --> H
    H --> I[Resultado Final Tipado]
    
    subgraph MessageBus [Barramento de Comunicação Compartilhado]
        D -.-> |Publica Dados| MessageBus
        E -.-> |Publica Código| MessageBus
        F -.-> |Consome Código| MessageBus
        G -.-> |Consome Código| MessageBus
    end
```

### 1.2 Dependências de Execução
O OMA é projetado para ser leve e rodar em qualquer ambiente Node.js 18+ (incluindo Serverless/AWS Lambda e ambientes de CI/CD), carregando apenas **3 dependências de runtime**:
1. `@anthropic-ai/sdk`
2. `openai`
3. `zod`

---

## 2. Configurações de Provedores de LLM (OpenAI & Anthropic)

O OMA adota uma estrutura de configuração estável (`AgentConfig`). Para alternar entre provedores, basta modificar os campos `provider` e `model` e fornecer as credenciais ambientais necessárias.

### 2.1 Configuração da Anthropic (Foco em Modelos Opus)
Para utilizar a Anthropic de maneira nativa, o OMA consome diretamente o SDK oficial da Anthropic. A chave de API deve ser configurada na variável de ambiente `ANTHROPIC_API_KEY`.

#### Exemplo de Configuração de Agente (Claude 3 Opus)
```typescript
import { OpenMultiAgent, type AgentConfig } from '@open-multi-agent/core';

const developerAgent: AgentConfig = {
  name: 'code-architect',
  provider: 'anthropic',
  model: 'claude-3-opus-20240229', // Modelo emblemático (flagship) para raciocínio complexo
  systemPrompt: 'Você é um arquiteto de software sênior encarregado de projetar estruturas de dados robustas.',
  tools: ['file_write', 'file_read'],
  // Configuração opcional para controlar parâmetros de inferência da Anthropic
  temperature: 0.2, // Baixa temperatura para manter a geração de código deterministicamente precisa
  maxTokens: 4000
};
```

#### Características de Execução do Claude 3 Opus no OMA:
* **Complexidade do Planejamento:** O OMA recomenda o uso do Claude 3 Opus (ou Claude 3.5 Sonnet) como modelo padrão para o agente **Coordenador/Planner**, pois a geração de um DAG de tarefas livre de ciclos (deadlocks) exige alta capacidade de raciocínio lógico e estruturação de JSON complexo.
* **Velocidade vs. Qualidade:** Embora o Opus tenha um tempo de resposta (Time to First Token) superior a modelos flash, o uso dele é priorizado na fase inicial (geração do DAG) e final (síntese e validação dos resultados). As tarefas intermediárias simples (folhas do DAG) são normalmente delegadas a modelos mais rápidos para economizar tempo e custo.

### 2.2 Configuração da OpenAI
A integração com os modelos da OpenAI utiliza a variável de ambiente `OPENAI_API_KEY`.

#### Exemplo de Configuração de Agente (GPT-4o)
```typescript
const reviewerAgent: AgentConfig = {
  name: 'security-reviewer',
  provider: 'openai',
  model: 'gpt-4o', // Modelo principal para análise de código e auditorias rápidas
  systemPrompt: 'Você é um engenheiro de segurança especializado em encontrar vulnerabilidades in APIs REST.',
  tools: ['grep', 'file_read'],
  temperature: 0.0 // Minimização máxima de alucinações para auditorias de segurança
};
```

### 2.3 Provedores Customizados e Compatibilidade OpenAI
Para utilizar endpoints customizados, servidores locais (como Ollama, LM Studio, vLLM) ou intermediários (como Groq, OpenRouter e Mistral), a configuração de provedor `'openai'` é estendida com os campos `baseURL` e `apiKey`:

```typescript
const localAgent: AgentConfig = {
  name: 'local-summarizer',
  provider: 'openai',
  model: 'qwen-2.5-coder', // Nome do modelo instalado no servidor local
  baseURL: 'http://localhost:11434/v1', // Aponta para a porta padrão do Ollama
  apiKey: 'ollama' // Credencial mock exigida pelo formato
};
```

### 2.4 Adaptação com Vercel AI SDK
Para obter compatibilidade imediata com mais de 60 modelos e plataformas suportadas pela comunidade, o OMA expõe a classe `AISdkAdapter`:

```typescript
import { AISdkAdapter } from '@open-multi-agent/core';
import { createOpenAI } from '@ai-sdk/openai';

const customProvider = createOpenAI({ apiKey: process.env.CUSTOM_API_KEY });

const bridgeAgent: AgentConfig = {
  name: 'bridge-agent',
  adapter: AISdkAdapter(customProvider('gpt-4o-mini')), // Encapsula o driver da Vercel
  tools: ['file_read']
};
```

---

## 3. Mecanismos de Controle e Confiabilidade (Safety)

Para mitigar custos descontrolados ("runs runaway"), loops infinitos de chamadas de ferramentas e garantir o controle humano, o OMA expõe cinco camadas de controle opcionais e configuráveis no runtime.

### 3.1 Controle de Custos (Cost Management)
1. **Model Routing:** Permite segmentar os agentes de acordo com seu custo. O desenvolvedor pode instanciar o orquestrador configurando o coordenador para usar o modelo flagship (ex: Claude 3 Opus) e definir modelos mais baratos (ex: `gpt-4o-mini` ou `claude-3-5-haiku`) como padrão para os agentes de execução secundários.
2. **`maxTokenBudget`:** Define um limite acumulado rígido de tokens consumidos durante toda a execução do time. Ao cruzar esse teto, o orquestrador interrompe a geração de novas tarefas no DAG e encerra a execução retornando uma falha controlada, evitando faturas abusivas.

```typescript
const orchestrator = new OpenMultiAgent({
  defaultModel: 'claude-3-5-sonnet-latest',
  maxTokenBudget: 150000 // Limite de 150k tokens (input + output combinados)
});
```

### 3.2 Controle Humano (Human-in-the-Loop)
* **`onPlanReady(plan)`:** Callback interceptor disparado imediatamente após a geração do DAG pelo coordenador, antes de qualquer tarefa ser executada. O desenvolvedor pode inspecionar o plano estruturado, exibi-lo em uma interface gráfica para aprovação do usuário, alterá-lo ou cancelá-lo.
* **`onApproval(round, pendingTasks)`:** Callback disparado ao final de cada rodada de execução (camadas do grafo). Retornar `false` cancela a execução de todas as tarefas subsequentes do DAG de forma limpa.

### 3.3 Verificação de Consenso (runConsensus)
O OMA implementa a arquitetura de **Proposer-Judge** (Propositor e Juiz). Uma tarefa crítica pode ser configurada para exigir consenso: o Agente A gera uma proposta de resposta, e o Agente B (que pode rodar um modelo diferente) deve analisar e aprovar. A tarefa só é dada como concluída no DAG se o consenso for atingido.

### 3.4 Proteções Contra Falhas (Guardrails)
* **Detecção de Loops (Loop Detection):** O runtime monitora a pilha de chamadas de ferramentas de cada agente. Se um agente invocar repetidamente a mesma ferramenta com os mesmos parâmetros ou emitir a mesma resposta sequencialmente, o OMA interrompe o agente com status `LOOP_DETECTED`.
* **Políticas de Retry Isoladas:** Se um nó do DAG falhar (ex: erro de rede do LLM ou falha de script), a falha é contida apenas naquele nó. O OMA executa retentativas isoladas de acordo com a política definida para a tarefa. Se a falha persistir, o nó entra em estado `FAILED`, bloqueando os nós descendentes, mas permitindo que ramos paralelos e independentes do DAG continuem rodando até o fim.

### 3.5 Observabilidade e Tracing
* **`onTrace(event)`:** Callback que envia todos os eventos de chamadas de ferramentas, requisições de LLM e transições de estado de nós diretamente para ferramentas de tracing (como OpenLIT, Langfuse ou Arize Phoenix).
* **Live Dashboard:** O comando da CLI `oma run --dashboard` gera uma página HTML estática local auto-contida. Ela reconstrói o grafo visual do DAG executado, mostrando tempos de resposta por nó, consumo de tokens discriminado por agente e logs completos de raciocínio.
