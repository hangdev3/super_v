# Migrador Windows → Linux

## Project Overview

Aplicativo desktop pequeno para reduzir o atrito de usuários que migram do Windows 11 para o Linux Mint Cinnamon. A primeira entrega reproduz o fluxo mais importante do `Win+V`: abrir um histórico de clipboard próximo ao ponteiro, escolher um item e deixá-lo disponível para colagem.

Nome de trabalho: `super_v`.

## Goals

- Oferecer um atalho global equivalente ao `Win+V`.
- Exibir o histórico como popup contextual próximo ao ponteiro do mouse.
- Registrar histórico de texto com limite, deduplicação e limpeza manual.
- Manter a lógica independente de X11, Wayland e Fyne por meio de interfaces.
- Não enviar dados para a rede nem persistir clipboard sensível por padrão.
- Criar uma base para outros atalhos familiares do Windows sem transformar o MVP em um “clone do Explorer”.

## Requirements

### User stories

- Como usuário migrando do Windows, quero pressionar `Super+V` para abrir o histórico sem procurar um ícone na bandeja, para manter o foco no editor atual.
- Como usuário, quero escolher um item do histórico e tê-lo no clipboard do sistema, para colá-lo no aplicativo que já estava usando.
- Como usuário, quero que novos textos copiados apareçam automaticamente no histórico, para não precisar cadastrá-los manualmente.
- Como usuário, quero limpar o histórico e desativar a persistência, para reduzir o risco de expor senhas e tokens.

### Acceptance criteria (EARS)

1. O sistema shall iniciar como um aplicativo residente sem exigir uma janela principal aberta.
2. When o usuário pressionar `Super+V`, o sistema shall consultar a posição atual do ponteiro e abrir o popup próximo a essa posição.
3. While o popup estiver aberto, o sistema shall mostrar os itens mais recentes primeiro e permitir seleção por mouse e teclado.
4. When o usuário selecionar um item de texto, o sistema shall escrever esse texto no clipboard do sistema e fechar o popup.
5. When um novo conteúdo textual diferente for detectado no clipboard, o sistema shall inseri-lo no topo do histórico.
6. If o conteúdo detectado for igual ao item mais recente, then o sistema shall evitar uma duplicata consecutiva.
7. If o histórico exceder o limite configurado, then o sistema shall remover os itens mais antigos.
8. When o usuário solicitar limpeza, o sistema shall remover todos os itens armazenados e atualizar o popup sem reiniciar o aplicativo.
9. If a integração gráfica ou de clipboard não estiver disponível, then o sistema shall continuar encerrando de forma controlada e informar uma ação corretiva clara.
10. If o ambiente for Wayland e a integração necessária não estiver disponível, then o sistema shall informar que o MVP requer X11/Cinnamon e não fingir que o atalho ou a posição foram registrados.
11. O sistema shall manter todo o histórico local, sem telemetria e sem chamadas de rede.
12. O sistema shall manter a persistência em disco desativada por padrão; quando ativada, os dados deverão ficar em um arquivo local de estado do usuário.

### MVP / fora do MVP

Incluído:

- Linux Mint Cinnamon em sessão X11.
- Go + Fyne.
- Histórico textual em memória.
- Popup contextual perto do ponteiro.
- Atalho global configurável, inicialmente `Super+V`.
- Seleção por mouse, Enter e setas.
- Limite de itens, deduplicação por conteúdo em todo o histórico e limpar tudo.
- Persistência JSON opcional.

Fora do MVP:

- Auto-paste no aplicativo que receber o foco externo, por injeção de teclado X11/XTEST.
- Imagens, HTML, arquivos e múltiplas seleções do clipboard.
- Wayland completo.
- Captura de tela, seletor de emoji e histórico compartilhado entre dispositivos.
- Sincronização em nuvem ou telemetria.

## Tech Stack

- Go 1.22+ (o ambiente atual usa Go 1.22.2).
- Fyne 2.8+ para interface desktop.
- `BurntSushi/xgb` para o adaptador global persistente no X11; os grabs ficam ativos até o encerramento e os eventos são encaminhados à UI.
- Adaptador de clipboard textual via utilitário disponível no ambiente (`xclip`/`xsel` no X11; `wl-clipboard` será uma implementação posterior para Wayland).
- Adaptador X11 para posição do ponteiro.
- JSON local opcional para persistência.
- Testes unitários com a biblioteca padrão `testing`; fakes para clipboard, hotkey, ponteiro e relógio.

## Architecture Diagram

```text
                       +--------------------------+
                       |        Cinnamon/X11      |
                       | hotkey | clipboard | X/Y  |
                       +------------+-------------+
                                    |
                 +------------------v------------------+
                 |        platform adapters            |
                 | hotkey | clipboard | pointer | fs   |
                 +------------------+------------------+
                                    |
                 +------------------v------------------+
                 |              core                   |
                 | history, dedupe, limit, selection   |
                 +------------------+------------------+
                                    |
                 +------------------v------------------+
                 |            Fyne UI                  |
                 | popup, keyboard navigation, errors  |
                 +-------------------------------------+
```

O core não poderá importar Fyne, X11, comandos externos ou variáveis de ambiente. Isso mantém os casos de uso testáveis em CI sem sessão gráfica.

## Repository Structure

```text
cmd/super-v/main.go
internal/core/              # regras de histórico e casos de uso
internal/platform/clipboard # leitura/escrita e monitoramento
internal/platform/hotkey    # registro do atalho global
internal/platform/pointer   # coordenadas do ponteiro
internal/platform/state     # persistência JSON opcional
internal/ui/fyne             # popup e navegação visual
docs/                        # decisões de produto e manual de instalação
```

## Data Models

```text
ClipItem {
  ID: string
  Text: string
  CreatedAt: time.Time
  Pinned: bool       # planejado; não usado no primeiro ciclo
}

History {
  Items: []ClipItem
  MaxItems: int
}
```

O primeiro ciclo trata apenas texto. O modelo já tem identidade própria para permitir, no futuro, tipos `image`, `html` e `file` sem depender de strings soltas.

## API Endpoints (planned)

Não haverá servidor HTTP no MVP. A API interna planejada é composta por casos de uso:

- `RecordText(text)`
- `ListRecent(limit)`
- `Select(id)`
- `Clear()`
- `SetPersistence(enabled)`
- `ShowAt(pointerPosition)`

Um comando CLI local poderá ser acrescentado depois para diagnóstico, sem escutar uma porta de rede.

## External Integrations

- X11/Cinnamon: atalho global, clipboard textual e posição do ponteiro.
- Fyne: janela popup, foco e navegação.
- `xclip` ou `xsel`: fallback operacional para leitura/escrita de texto no X11.
- Wayland/XDG Desktop Portal: futura implementação de `GlobalShortcuts`; o MVP deve detectar e explicar a limitação em vez de assumir que APIs X11 funcionarão.

## Environment Variables

Nenhuma variável obrigatória.

Configuração opcional futura:

- `SUPERV_HOTKEY`: atalho global, default `Super+V`.
- `SUPERV_MAX_ITEMS`: default `50`.
- `SUPERV_PERSIST_HISTORY`: default `false`.
- `SUPERV_STATE_FILE`: caminho opcional do JSON local.

## Deployment Target

- Distribuição inicial: binário local para Linux Mint Cinnamon/X11.
- Entrega posterior: pacote `.deb` e arquivo `.desktop` com autostart opcional.
- Requisitos de runtime do MVP: sessão gráfica X11, `xclip` ou `xsel` e bibliotecas de runtime exigidas pelo Fyne.
- CI deverá executar testes do core e build; testes visuais/atalho serão smoke tests manuais em uma sessão Cinnamon/X11.

## Security and Privacy

- Histórico somente local.
- Persistência desligada por padrão.
- A interface deve oferecer “Limpar histórico”.
- Não registrar textos de clipboard em logs.
- Limitar tamanho máximo de cada entrada para evitar consumo acidental de memória/disco.
- Documentar que clipboard managers podem capturar senhas e tokens; o usuário deve poder pausar o monitoramento em uma etapa posterior.

## Decision Log

- AD-001 (2026-09-29): começar por Linux Mint Cinnamon/X11. O posicionamento arbitrário e o registro de hotkey são mais previsíveis nesse ambiente; Wayland terá adaptadores próprios. Status: ativa.
- AD-002 (2026-09-29): usar Fyne apenas na camada visual. Hotkey, clipboard, ponteiro e persistência ficam atrás de interfaces para manter o core testável. Status: ativa.
- AD-003 (2026-09-29): histórico textual, seleção que atualiza o clipboard e auto-paste best-effort no X11 são o MVP; Wayland permanece sem injeção de teclado até existir um adaptador próprio. Status: ativa.
- AD-004 (2026-09-29): persistência desligada por padrão por risco de guardar segredos. Status: ativa.
- AD-005 (2026-09-29): não criar servidor HTTP nem telemetria. O produto é local-first. Status: ativa.
- AD-006 (2026-09-30): no X11, registrar `Super+V` também com as combinações de `Caps Lock` e `Num Lock`, porque esses modificadores fazem parte do estado da tecla e podem impedir um grab exato de `Mod4`. Status: ativa.
- AD-007 (2026-09-30): manter o popup compacto (342x300), com ação de limpeza iconográfica e separadores visuais por item para preservar o foco no aplicativo em uso. Status: ativa.
- AD-008 (2026-09-30): o popup será dispensado quando perder o foco X11 por interação externa; mover o ponteiro para fora não encerra a janela. A janela será identificada preferencialmente pelo `_NET_WM_PID` do processo e suas decorações serão removidas via `_MOTIF_WM_HINTS` quando a sessão X11 permitir. Status: ativa.
- AD-009 (2026-09-30): eventos repetidos do `Super+V` durante o auto-repeat do teclado serão ignorados por uma janela de 250 ms, evitando múltiplas aberturas concorrentes. Status: ativa.
- AD-010 (2026-09-30): o popup deve permanecer visível por pelo menos 800 ms antes de avaliar ponteiro e foco, evitando que o próprio evento `Super+V` provoque o fechamento imediato. Status: ativa.
- AD-011 (2026-09-30): o processo registra `SIGINT` e `SIGTERM` para encerrar o loop do Fyne e liberar os recursos de hotkey, ponteiro e janela antes de sair. Status: ativa.
- AD-012 (2026-09-30): a saída do ponteiro não é um gesto de fechamento; a seleção de um item fecha após copiar, e o clique em outra janela é detectado pela perda de foco X11. Status: ativa.
- AD-013 (2026-09-30): a janela do popup deve ser localizada prioritariamente pela propriedade X11 `_NET_CLIENT_LIST`, evitando falhas em gerenciadores que reparentam janelas; a árvore da raiz permanece como fallback. Status: ativa.
- AD-014 (2026-09-30): após selecionar um item, o popup permanece armado; ao perder o foco para uma janela externa, o adaptador X11 envia `Ctrl+V` e então fecha o popup. Como X11 não expõe genericamente o tipo do widget focado, o auto-paste é best-effort e fica limitado a sessões com XTEST. Status: ativa.
- AD-015 (2026-09-30): a inicialização do hotkey aguardava 100 ms após os grabs X11 e a abertura do popup usava `fyne.DoAndWait`, evitando que o primeiro `Super+V` fosse perdido durante a inicialização do loop gráfico. Status: superseded by AD-016.
- AD-016 (2026-09-30): o hotkey usa grabs X11 persistentes via XGB, filtrando apenas `KeyPress` e liberando os grabs somente no encerramento; isso elimina a janela entre `ungrab` e novo `grab` que permitia o primeiro `V` chegar ao editor. Status: ativa.
- AD-014 (2026-09-30): uma entrada textual que já exista em qualquer posição do histórico não deve gerar uma nova linha; a comparação é feita pelo conteúdo exato. Status: ativa.

## Definition of Done — primeiro ciclo

- Os testes do core cobrem inserção, deduplicação, limite, seleção e limpeza.
- A aplicação registra o hotkey e abre um popup Fyne na posição do ponteiro em Cinnamon/X11.
- Um clique/Enter copia o item escolhido para o clipboard.
- O binário falha com diagnóstico útil quando executado sem X11 ou sem utilitário de clipboard.
- `go test ./...` passa e o build do Linux é reproduzível.
