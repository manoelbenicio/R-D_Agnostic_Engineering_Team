# RELATÓRIO DE DIAGNÓSTICO READ-ONLY & PREFLIGHT GATE 0 — ORQ-26 (BOTÕES DO PAINEL DE CHAT)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (General-TL) & Owner
- **Card Alvo:** ORQ-26 (UUID: `e966922d-c6a5-4812-87bb-8b9576ccbc60`)
- **Papel:** Único Executor / Single Writer Designado
- **Data UTC:** 2026-07-28T16:09:45Z
- **ETA Diagnóstico:** 20-40m (Concluído em 12m)
- **ETA Implementação:** 45-90m

---

## 1. Confirmação de Causa Raiz & FILES_LOCKED

- **Arquivo Exclusivo de Escrita (`FILES_LOCKED`):**
  [chat-window.tsx](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/packages/views/chat/components/chat-window.tsx#L694-L749)
- **Causa Raiz Medida no Código:**
  - Nas linhas 694-749 de `chat-window.tsx`, os ícones de ação (`<Plus />`, `<Minimize2 />`/`<Maximize2 />`, `<Minus />`) foram passados como filhos diretos de `<TooltipTrigger>` fora da prop `render={<Button ... />}`.
  - Na biblioteca `@base-ui/react/tooltip`, quando a prop `render` é fornecida ao `TooltipTrigger`, a renderização utiliza o elemento passado em `render`. Filhos externos passados fora de `render` são ignorados, resultando em botões vazios e sem ícone no cabeçalho da janela de chat.
- **Diferenciação Estrita:** Esta regressão no painel UI de chat **NÃO se confunde** com o antigo gate de CI homônimo.

---

## 2. Plano de Implementação Promovido

Ajustar a estrutura JSX nas linhas 694-749 de `chat-window.tsx` para incorporar os ícones diretamente dentro do elemento `<Button>` da prop `render`:

```tsx
// 1. Novo Chat (+):
<TooltipTrigger
  render={
    <Button
      variant="ghost"
      size="icon-sm"
      className="rounded-full text-muted-foreground"
      onClick={handleNewChat}
    >
      <Plus />
    </Button>
  }
/>

// 2. Expandir / Restaurar:
<TooltipTrigger
  render={
    <Button
      variant="ghost"
      size="icon-sm"
      className="text-muted-foreground"
      onClick={toggleExpand}
    >
      {isExpanded || isAtMax ? <Minimize2 /> : <Maximize2 />}
    </Button>
  }
/>

// 3. Minimizar (-):
<TooltipTrigger
  render={
    <Button
      variant="ghost"
      size="icon-sm"
      className="text-muted-foreground"
      onClick={handleMinimize}
    >
      <Minus />
    </Button>
  }
/>
```

---

## 3. Preflight Gate 0 (Status & Garantias)

- [x] **Agente Executor Único:** `Agy-P0-A8` é o único escritor autorizativo de `chat-window.tsx`.
- [x] **FILES_LOCKED:** `packages/views/chat/components/chat-window.tsx` bloqueado e isolado.
- [x] **Worktree Limpo:** `git status` limpo antes da edição.
- [x] **Zero Mutações não Autorizadas:** Nenhuma escrita no banco de dados, repositório remoto ou board Kanban antes do sinal.
