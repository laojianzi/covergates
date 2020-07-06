import { Route, NavigationGuardNext } from 'vue-router';
import { Store } from 'vuex';
import { RootState, Actions, State } from '@/store';

type RouteHandler = (to: Route, from: Route, next: NavigationGuardNext) => void;

export function fetchCurrentRepository(store: Store<RootState>): RouteHandler {
  return (to, from, next) => {
    store.dispatch(Actions.CHANGE_CURRENT_REPOSITORY, to.params)
      .then(() => {
        if ((store.state as State).repository.current) {
          store.dispatch(
            Actions.FETCH_REPORT_CURRENT,
            (store.state as State).repository.current?.ReportID);
        }
      });
    next();
  };
}

// TODO: implement fetch source code
export function fetchReportSource(store: Store<RootState>): RouteHandler {
  return (to, from, next) => {
    next();
  };
}