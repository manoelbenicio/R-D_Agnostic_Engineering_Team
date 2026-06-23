# PRD — Agent Squad (Multi-Agent Orchestrator) — Architecture & Configuration Mappings

Este documento apresenta o mapeamento detalhado da arquitetura, fluxo de roteamento de intenções e padrões de coordenação do framework **Agent Squad** (https://github.com/2fastlabs/agent-squad), anteriormente conhecido como **AWS Multi-Agent Orchestrator**, com foco especial no `SupervisorAgent` e nos classificadores da **Anthropic** e **OpenAI**.

---

## 1. Visão Geral da Arquitetura do Agent Squad

O **Agent Squad** é um framework modular e extensível (disponível para Python e TypeScript) desenvolvido inicialmente pela AWS Labs para gerenciar e rotear conversações complexas e de múltiplos turnos envolvendo uma equipe de agentes de inteligência artificial.

### 1.1 Roteamento Dinâmico por Classificação
Diferente do modelo centrado em grafos de tarefas fixos, o Agent Squad utiliza uma arquitetura baseada em **Classificadores (Classifiers)**. O fluxo operacional padrão segue quatro etapas:

1. **Entrada do Usuário:** O sistema recebe a mensagem do usuário juntamente com o identificador da sessão (`sessionId`).
2. **Classificação (Roteamento):** O Classificador analisa a mensagem, o histórico acumulado da sessão (armazenado de forma persistente) e as descrições estruturadas de cada agente disponível no sistema. Ele decide qual agente é o mais qualificado para tratar a mensagem.
3. **Execução Especializada:** O agente selecionado processa a mensagem e gera a resposta.
4. **Persistência de Contexto:** O orquestrador salva a interação, atualiza a memória compartilhada da sessão e retorna o resultado ao usuário.

---

## 2. O Sistema de Classificadores (Classifiers)

Os classificadores são o coração do roteamento de conversas no Agent Squad. O framework fornece implementações nativas prontas para uso (out-of-the-box) para OpenAI e Anthropic, permitindo também a criação de roteadores customizados.

### 2.1 OpenAIClassifier
O `OpenAIClassifier` utiliza os modelos de chat GPT da OpenAI para classificar a intenção do usuário e determinar o agente de destino.

#### Parâmetros e Opções de Configuração (TypeScript & Python)
A inicialização exige as credenciais da OpenAI (`OPENAI_API_KEY`) e aceita as seguintes opções de configuração (`OpenAIClassifierOptions`):
* `api_key`: Chave de acesso da OpenAI.
* `model_id`: O identificador do modelo (Ex: `'gpt-4o'`, `'gpt-4o-mini'`).
* `inference_config`: Objeto contendo parâmetros de inferência fine-grained:
  * `temperature`: Controla a criatividade da classificação (recomendado `0.0` para manter decisões de roteamento deterministicamente estáveis).
  * `max_tokens`: O limite máximo de tokens para a resposta de classificação.
  * `top_p`: Configuração de amostragem por núcleo.

```typescript
// Exemplo de configuração em TypeScript
import { OpenAIClassifier, OpenAIClassifierOptions, MultiAgentOrchestrator } from 'agent-squad';

const openAiClassifier = new OpenAIClassifier({
  apiKey: process.env.OPENAI_API_KEY!,
  modelId: 'gpt-4o',
  inferenceConfig: {
    temperature: 0.0,
    maxTokens: 500,
    topP: 0.9
  }
});

const orchestrator = new MultiAgentOrchestrator({ classifier: openAiClassifier });
```

### 2.2 AnthropicClassifier
O `AnthropicClassifier` utiliza modelos Claude da Anthropic para executar a lógica de classificação.

#### Parâmetros e Opções de Configuração (TypeScript & Python)
Configurado a partir de `AnthropicClassifierOptions`:
* `api_key`: Chave de acesso da Anthropic (`ANTHROPIC_API_KEY`).
* `model_id`: O identificador do modelo. Ex: `'claude-3-5-sonnet-latest'` ou o modelo emblemático de raciocínio de alta fidelidade **`'claude-3-opus-20240229'`**.

```python
# Exemplo de configuração em Python
from agent_squad.classifiers import AnthropicClassifier, AnthropicClassifierOptions
from agent_squad.orchestrator import AgentSquad

anthropic_classifier = AnthropicClassifier(AnthropicClassifierOptions(
    api_key=os.environ.get("ANTHROPIC_API_KEY"),
    model_id="claude-3-opus-20240229" # Foco em modelos Opus para raciocínio analítico máximo
))

orchestrator = AgentSquad(classifier=anthropic_classifier)
```

### 2.3 Classificadores Customizados (Custom Classifier)
Para implementar lógicas de roteamento proprietárias (como classificação baseada em correspondência de palavras-chave, regex ou buscas vetoriais em bancos de dados semânticos), os desenvolvedores podem herdar diretamente da classe abstrata `Classifier` e sobrescrever o método `classify()`:

```typescript
import { Classifier, ClassifierResult, AgentInfo } from 'agent-squad';

export class CustomClassifier extends Classifier {
  async classify(userInput: string, agents: AgentInfo[]): Promise<ClassifierResult> {
    // Lógica personalizada de roteamento
    const targetAgent = agents.find(agent => userInput.toLowerCase().includes(agent.name.toLowerCase()));
    
    return {
      selectedAgent: targetAgent || agents[0], // Fallback para o primeiro agente
      confidence: 0.95,
      explanation: 'Matches key terms in custom routing rules.'
    };
  }
}
```

---

## 3. SupervisorAgent: Coordenação e Colaboração

O `SupervisorAgent` é a principal ferramenta de coordenação do Agent Squad, permitindo transitar de um modelo simples de roteamento para uma colaboração avançada entre agentes.

### 3.1 Arquitetura "Agent-as-Tools"
A arquitetura do `SupervisorAgent` baseia-se em tratar outros agentes especializados como se fossem ferramentas (`tools`) que o supervisor pode invocar livremente.

* **Supervisor (Lead Agent):** O agente líder (geralmente configurado com um modelo robusto como Claude 3 Opus ou GPT-4o) que planeja a estratégia para responder ao usuário.
* **Squad (Sub-Agentes):** Agentes especialistas que possuem escopos fechados e ferramentas específicas (ex: um agente desenvolvedor, um agente escritor de SQL, um analista de dados).

```
                 ┌──────────────────┐
                 │    User Query    │
                 └────────┬─────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │   SupervisorAgent   │
               └─┬─────────────────┬─┘
                 │                 │
      (Invoca como Ferramenta)  (Invoca como Ferramenta)
                 │                 │
                 ▼                 ▼
         ┌───────────────┐ ┌───────────────┐
         │ Agent A (SQL) │ │ Agent B (Plot)│
         └───────────────┘ └───────────────┘
```

### 3.2 Fluxo de Processamento e Execução
1. O `SupervisorAgent` recebe a pergunta complexa do usuário (ex: *"Busque as vendas de ontem no banco e gere um gráfico"*).
2. O Supervisor percebe que não consegue resolver a tarefa sozinho. Ele gera uma sub-query formatada como chamada de ferramenta direcionada ao *Agent A (SQL)*.
3. O *Agent A* executa a consulta SQL e retorna os dados brutos ao Supervisor.
4. O Supervisor consome a resposta do *Agent A* e formata uma segunda sub-query direcionada ao *Agent B (Plot)* contendo os dados e pedindo a plotagem do gráfico.
5. O *Agent B* retorna a imagem/dados do gráfico ao Supervisor.
6. O Supervisor consolida os outputs recebidos dos especialistas, escreve a resposta final resumida e entrega ao usuário.

### 3.3 Configuração e Armazenamento (DynamoDbChatStorage)
Para manter o histórico e o contexto sincronizados entre todos os agentes participantes da coordenação, o framework utiliza adaptadores de armazenamento persistente. No ambiente corporativo/AWS, a classe `DynamoDbChatStorage` é recomendada para persistência rápida de sessões:

```typescript
import { DynamoDbChatStorage, MultiAgentOrchestrator, SupervisorAgent } from 'agent-squad';

// Instanciação do DynamoDB para persistência de memória
const dynamoDbStorage = new DynamoDbChatStorage(
  process.env.MULTI_AGENT_TABLE_NAME!, // Nome da tabela
  process.env.AWS_REGION! // Região AWS
);

// Instanciação do SupervisorAgent
const supervisor = new SupervisorAgent({
  name: 'manager-agent',
  description: 'Coordena equipes de desenvolvimento e infraestrutura para realizar deploys.',
  leadAgent: myLeadOpenAiAgent, // Agente líder
  team: [developerAgent, infraAgent], // Equipe de sub-agentes
  storage: dynamoDbStorage,
  trace: true // Habilita rastreamento completo das delegações
});

const orchestrator = new MultiAgentOrchestrator({
  storage: dynamoDbStorage
});

orchestrator.addAgent(supervisor);
```

---

## 4. Observabilidade e Tracing

A depuração e o monitoramento em produção de fluxos multi-agente coordenados pelo `SupervisorAgent` são viabilizados por meio do mecanismo de **Callbacks de Observabilidade**.

### 4.1 Ciclo de Vida de Callbacks
O desenvolvedor pode escutar eventos em tempo real do ciclo de vida das chamadas dos agentes assinando ouvintes no orquestrador:

* `onAgentCallStart`: Disparado no instante em que o classificador ou o supervisor delega uma tarefa a um agente específico. Fornece metadados do agente de destino e o payload de entrada.
* `onAgentCallEnd`: Disparado no encerramento do processamento do agente. Retorna o texto de saída, tempo decorrido de execução e métricas de consumo de tokens do modelo.
* `onAgentCallError`: Captura exceções e falhas de runtime do LLM ou falhas de chamadas de ferramentas.

Isso permite registrar métricas detalhadas de custos por modelo (discriminando o uso de Claude 3 Opus vs. modelos GPT), alimentar dashboards executivos e auditar a qualidade das decisões tomadas pelo `SupervisorAgent`.
