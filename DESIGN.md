# DESIGN.md — Réflexion Architecturale

## 1. Découpage et frontières d'interface

Le projet suit une **architecture hexagonale légère** avec un découpage par packages isolant le domaine de l'infrastructure :
- `domain` : Centre de l'application. Contient les types métier (`Batch`, `CheckResult`) et les définitions d'interfaces (`Checker`, `Store`). 
- `api` (handler REST), `checker` (HTTP client), `store` (mémoire) et `pool` dépendent de `domain`, **jamais l'inverse**.
- **Frontières d'interface** : Placées exactement entre la logique de coordination (le handler/pool) et l'accès I/O (`Store` pour la donnée, `Checker` pour le réseau). Cela permet d'isoler les side-effects : le worker pool est testable de manière 100% déterministe en lui injectant un `MockChecker`.

## 2. Modèle de concurrence et échecs partiels

Le worker pool utilise un pattern **fan-out / fan-in** borné :
- **Taille du pool** : Strictement limitée à `concurrency` (par défaut 8). Lancer une goroutine par URL saturerait la table des descripteurs de fichiers (sockets) et écroulerait le service sous forte charge.
- **Bufferisation `jobs`** : Bufferisé à `concurrency`. Permet au producteur de pousser des tâches en avance sans allouer de mémoire pour des milliers d'URLs (`len(urls)` serait un gaspillage).
- **Bufferisation `results`** : Bufferisé à `len(urls)`. Chaque check produit exactement un résultat. Cette taille garantit que les workers ne bloquent **jamais** en écriture, éliminant tout risque de deadlock au moment du fan-in.
- **Échecs partiels** : Si une URL échoue (timeout, 500, erreur DNS), le worker renvoie un `CheckResult` avec `ok: false` et le message d'erreur. Le processus global continue. La fonction `domain.Summarize` agrège ensuite ces succès/échecs (`up`/`down`).

## 3. Prévention des fuites de goroutines

Le risque principal de fuite survient si l'émetteur bloque sur un channel `jobs` plein, ou si les workers bloquent sur un `results` non lu, particulièrement lors d'une annulation de `context`.
**Contre-mesures concrètes dans notre code (`worker.go`) :**
1. **Goroutine d'émission (`jobs <- url`)** : Écoute `<-ctx.Done()` dans un `select`. Si le contexte expire, la boucle s'arrête et exécute `defer close(jobs)`, libérant ainsi les workers.
2. **Buffer `results`** : Comme sa capacité égale le nombre total de tâches, un worker qui tente de remonter une erreur après annulation du contexte ne restera jamais bloqué.
3. **`sync.WaitGroup`** : L'utilisation de `defer wg.Done()` dans chaque worker garantit le décrément du compteur, même en cas de panic. La goroutine de collecte attend `wg.Wait()` avant de faire `close(results)`, évitant de fermer le channel alors que des workers écrivent encore.

## 4. Stratégie de gestion des erreurs

La gestion s'appuie sur les standards idiomatiques de Go 1.13+ :
- **Erreurs sentinelles** : `ErrBatchNotFound` pour les cas prévisibles.
- **Types personnalisés** : `ValidationError` implémente `error` et encapsule le champ fautif.
- **Wrapping** : Dans le Store, l'erreur est enrichie de son contexte : `fmt.Errorf("... : %w", ErrBatchNotFound)`.
- **Lien avec l'API HTTP** : Dans `handler.go`, le routeur utilise `errors.Is(err, domain.ErrBatchNotFound)` pour renvoyer une **404**, et `errors.As(err, &valErr)` pour mapper une erreur de validation en **400 Bad Request**. Toute autre erreur tombe en **500 Internal Server Error**.

## 5. Philosophie Go : Comparaison et Limites

**Pourquoi Go est un bon choix ici (3 arguments issus du code) :**
1. **Concurrence native et lisible** : Pas de `async/await` contagieux comme en Python ou Rust, ni de lourds `ExecutorService` Java. Le mot-clé `go` et les `channels` permettent d'écrire un worker pool fan-out/fan-in en 60 lignes très expressives.
2. **Déploiement statique et léger** : Le microservice compile en un unique binaire natif autonome (`go build`), sans JVM ni interpréteur, idéal pour un déploiement cloud rapide avec une empreinte mémoire minime (grâce à notre pool borné).
3. **Typage structurel des interfaces** : En Java, la classe doit déclarer `implements Checker`. En Go, notre mock implémente l'interface implicitement. Cela permet de définir les interfaces là où elles sont utilisées (`domain`), garantissant un couplage très faible.

**Une limite ressentie de Go :**
Le système de gestion d'erreurs, bien que clair, peut être verbeux. Répéter la vérification `if err != nil` à chaque étape de la couche HTTP (lecture JSON, parsing URL, validation de chaque champ, sauvegarde Store) alourdit considérablement le code métier comparativement à l'utilisation d'exceptions encapsulées ou de combinateurs comme `Result` (en Rust).
