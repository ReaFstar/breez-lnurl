package nwc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/breez/breez-lnurl/persist"
	nwc "github.com/breez/breez-lnurl/persist/nwc"
	"github.com/breez/lspd/lightning"
	"github.com/gorilla/mux"
)

type NostrEventsRouter struct {
	store   *persist.Store
	manager *NostrManager
	rootURL *url.URL
}

func RegisterNostrEventsRouter(router *mux.Router, rootURL *url.URL, store *persist.Store, cleanupService *nwc.CleanupService) {
	NostrEventsRouter := &NostrEventsRouter{
		store:   store,
		manager: NewNostrManager(store),
		rootURL: rootURL,
	}
	NostrEventsRouter.manager.Start()
	router.HandleFunc("/nwc/{walletPubkey}", NostrEventsRouter.Register).Methods("POST")
	router.HandleFunc("/nwc/{walletPubkey}", NostrEventsRouter.Unregister).Methods("DELETE")
}

type RegisterNostrEventsRequest struct {
	WebhookUrl string   `json:"webhookUrl"`
	UserPubkey string   `json:"userPubkey"`
	AppPubkey  string   `json:"appPubkey"`
	Relays     []string `json:"relays"`
	Signature  string   `json:"signature"`
}

func (w *RegisterNostrEventsRequest) Verify(pubkey string) error {
	messageToVerify := fmt.Sprintf("%v-%v-%v-%v", w.WebhookUrl, w.UserPubkey, w.AppPubkey, w.Relays)
	verifiedPubkey, err := lightning.VerifyMessage([]byte(messageToVerify), w.Signature)
	if err != nil {
		return err
	}
	if pubkey != hex.EncodeToString(verifiedPubkey.SerializeCompressed()) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

/*
Register adds a registration for a given pubkey, overwriting it if already present
*/
func (s *NostrEventsRouter) Register(w http.ResponseWriter, r *http.Request) {
	var registerRequest RegisterNostrEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&registerRequest); err != nil {
		log.Printf("json.NewDecoder.Decode error: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	params := mux.Vars(r)
	walletPubkey, ok := params["walletPubkey"]
	if !ok {
		http.Error(w, "invalid wallet pubkey", http.StatusBadRequest)
		return
	}

	if err := registerRequest.Verify(walletPubkey); err != nil {
		log.Printf("failed to verify registration request: %v", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	err := s.store.Nwc.Set(r.Context(), nwc.Webhook{
		UserPubkey: registerRequest.UserPubkey,
		Url:        registerRequest.WebhookUrl,
		AppPubkey:  registerRequest.AppPubkey,
		Relays:     registerRequest.Relays,
	})
	if err != nil {
		log.Printf("failed to persist nwc details: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("registration added: pubkey:%v\n", registerRequest.UserPubkey)
	w.Write([]byte("Pubkey registered successfully"))
}

type UnregisterNostrEventsRequest struct {
	Time       int64  `json:"time"`
	UserPubkey string `json:"userPubkey"`
	AppPubkey  string `json:"appPubkey"`
	Signature  string `json:"signature"`
}

func (w *UnregisterNostrEventsRequest) Verify(pubkey string) error {
	messageToVerify := fmt.Sprintf("%v-%v-%v", w.Time, w.UserPubkey, w.AppPubkey)
	verifiedPubkey, err := lightning.VerifyMessage([]byte(messageToVerify), w.Signature)
	if err != nil {
		return err
	}
	if pubkey != hex.EncodeToString(verifiedPubkey.SerializeCompressed()) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (s *NostrEventsRouter) Unregister(w http.ResponseWriter, r *http.Request) {
	var req UnregisterNostrEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("json.NewDecoder.Decode error: %v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	params := mux.Vars(r)
	walletPubkey, ok := params["walletPubkey"]
	if !ok {
		http.Error(w, "invalid pubkey", http.StatusBadRequest)
		return
	}

	if err := req.Verify(walletPubkey); err != nil {
		log.Printf("failed to verify registration request: %v", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	err := s.store.Nwc.Delete(r.Context(), req.UserPubkey, req.AppPubkey)
	if err != nil {
		log.Printf("failed to delete nwc webhook: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("registration deleted: pubkey:%v\n", req.UserPubkey)
	w.Write([]byte("Pubkey unregistered successfully"))
}
