import AppDispatcher from '../appDispatcher';
import AppConstants from '../appConstants';

export default {
  updateFilter: (filters) => {
    AppDispatcher.handleAction({
      actionType: AppConstants.ActionTypes.UPDATE_FILTER,
      data: filters
    });
  },
  resetFilters: () => {
    AppDispatcher.handleAction({
      actionType: AppConstants.ActionTypes.RESET_FILTERS
    });
  },
  clearFilters: () => {
    AppDispatcher.handleAction({
      actionType: AppConstants.ActionTypes.CLEAR_FILTERS
    });
  }
};
